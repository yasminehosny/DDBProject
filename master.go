package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:12345678@tcp(localhost:3306)/")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS history_query")
	if err != nil {
		log.Fatal("Failed to create database:", err)
	}
	fmt.Println("Database history_query created or already exists")

	_, err = db.Exec("USE history_query")
	if err != nil {
		log.Fatal("Failed to select database:", err)
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS queries (
        id INT AUTO_INCREMENT PRIMARY KEY,
        query_text TEXT,
        slave_ip VARCHAR(45),
        database_name VARCHAR(255),
        table_name VARCHAR(255),
        executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    `
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	fmt.Println("Table queries created or already exists")
	http.HandleFunc("/queries_log", queriesLogHandler)

	http.HandleFunc("/create_database", createDatabaseHandler)
	http.HandleFunc("/drop_database", dropDatabaseHandler)
	http.HandleFunc("/create_table", createTableHandler)
	http.HandleFunc("/drop_table", dropTableHandler)
	http.HandleFunc("/execute_query", executeQueryHandler)

	fmt.Println("Master API server running on port 8000")
	http.ListenAndServe(":8000", nil)
}

type DBRequest struct {
	DBName string `json:"dbname"`
}

type TableRequest struct {
	DBName  string   `json:"dbname"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

func createDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	var req DBRequest
	json.NewDecoder(r.Body).Decode(&req)

	var exists bool
	query := fmt.Sprintf("SELECT COUNT(*) > 0 FROM information_schema.schemata WHERE schema_name = '%s'", req.DBName)
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if exists {
		w.Write([]byte("Database already exists"))
		return
	}

	createQuery := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", req.DBName)
	_, err = db.Exec(createQuery)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Database created"))
}

func dropDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	var req DBRequest
	json.NewDecoder(r.Body).Decode(&req)

	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", req.DBName)
	_, err := db.Exec(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Database dropped"))
}

func createTableHandler(w http.ResponseWriter, r *http.Request) {
	var req TableRequest
	json.NewDecoder(r.Body).Decode(&req)

	useDB := fmt.Sprintf("USE %s", req.DBName)
	_, err := db.Exec(useDB)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var exists bool
	query := fmt.Sprintf("SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'", req.DBName, req.Table)
	err = db.QueryRow(query).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if exists {
		w.Write([]byte("Table already exists"))
		return
	}

	columns := ""
	for i, col := range req.Columns {
		if i > 0 {
			columns += ", "
		}
		columns += col
	}

	if columns == "" {
		http.Error(w, "Columns cannot be empty", 400)
		return
	}

	createQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", req.Table, columns)
	_, err = db.Exec(createQuery)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Table created"))
}

func dropTableHandler(w http.ResponseWriter, r *http.Request) {
	var req TableRequest
	json.NewDecoder(r.Body).Decode(&req)

	useDB := fmt.Sprintf("USE %s", req.DBName)
	_, err := db.Exec(useDB)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", req.Table)
	_, err = db.Exec(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("Table dropped"))
}

func executeQueryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DBName string `json:"dbname"`
		Query  string `json:"query"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), 400)
		return
	}

	useDB := fmt.Sprintf("USE %s", req.DBName)
	_, err = db.Exec(useDB)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ip := strings.Split(r.RemoteAddr, ":")[0]
	fmt.Printf("Master received a query from slave at IP: %s\n", ip)

	if !isAllowedSlaveQuery(req.Query) {
		http.Error(w, "Query type not allowed for slaves", 403)
		return
	}

	tableName := extractTableName(req.Query) // استخراج اسم الجدول

	logSlaveQuery(req.Query, ip, req.DBName, tableName)

	if isSelectQuery(req.Query) {
		rows, err := db.Query(req.Query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		results := []map[string]interface{}{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePointers := make([]interface{}, len(columns))
			for i := range values {
				valuePointers[i] = &values[i]
			}

			err := rows.Scan(valuePointers...)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			rowMap := make(map[string]interface{})
			for i, colName := range columns {
				val := values[i]
				rowMap[colName] = val
			}
			results = append(results, rowMap)
		}

		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		response := struct {
			Results []map[string]interface{} `json:"results"`
		}{
			Results: results,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		result, err := db.Exec(req.Query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		io.WriteString(w, fmt.Sprintf("Query executed successfully, affected %d rows", rowsAffected))
	}
}

func isAllowedSlaveQuery(query string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	// السماح فقط بالاستعلامات التالية للسلاف:
	// SELECT, INSERT, UPDATE, DELETE فقط
	if strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "INSERT") ||
		strings.HasPrefix(trimmed, "UPDATE") ||
		strings.HasPrefix(trimmed, "DELETE") {
		return true
	}
	return false
}

func isSelectQuery(query string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(query))
	return strings.HasPrefix(trimmed, "SELECT")
}

func extractTableName(query string) string {
	query = strings.ToUpper(query)
	tokens := strings.Fields(query)
	for i, token := range tokens {
		if token == "FROM" || token == "INTO" || token == "UPDATE" || token == "JOIN" {
			if i+1 < len(tokens) {
				tbl := strings.Trim(tokens[i+1], ";,")
				return tbl
			}
		}
	}
	return ""
}

func logSlaveQuery(query, ip, dbName, tableName string) {
	insertSQL := `
    INSERT INTO history_query.queries (query_text, slave_ip, database_name, table_name)
    VALUES (?, ?, ?, ?)
    `
	_, err := db.Exec(insertSQL, query, ip, dbName, tableName)
	if err != nil {
		fmt.Printf("Failed to log query: %v\n", err)
	}
}
func queriesLogHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, query_text, slave_ip, database_name, table_name, executed_at FROM history_query.queries ORDER BY executed_at DESC LIMIT 100")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type QueryLog struct {
		ID           int    `json:"id"`
		QueryText    string `json:"query_text"`
		SlaveIP      string `json:"slave_ip"`
		DatabaseName string `json:"database_name"`
		TableName    string `json:"table_name"`
		ExecutedAt   string `json:"executed_at"`
	}

	var logs []QueryLog
	for rows.Next() {
		var log QueryLog
		err := rows.Scan(&log.ID, &log.QueryText, &log.SlaveIP, &log.DatabaseName, &log.TableName, &log.ExecutedAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
