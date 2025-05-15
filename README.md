
# 🗄️ Distributed Master-Slave Database System (Go + Streamlit)

This project implements a distributed SQL execution system using a **master-slave architecture**. The system is written in **Go** for the backend, and **Streamlit** for the web user interfaces.

---

## 📦 Components

### 1️⃣ Master Server (`master.go`)
- Written in Go.
- Hosts REST APIs to:
  - Create/Drop Databases.
  - Create/Drop Tables.
  - Execute validated queries (from slaves only).
  - Log all slave queries into `history_query.queries` table.
- Logs metadata such as:
  - IP address of slave.
  - Database and table names.
  - Timestamp.

### 2️⃣ Master Web Interface (`master.py`)
- Built with Streamlit.
- Allows the user to:
  - Create/Drop Databases.
  - Create/Drop Tables (by specifying column definitions).
  - View the query logs from slaves.

### 3️⃣ Slave Server (`slave.go`)
- Written in Go.
- Exposes a `/query` endpoint.
- Accepts JSON payloads with:
  - Database name.
  - SQL query (SELECT/INSERT/UPDATE/DELETE only).
- Forwards the query to the master server’s `/execute_query` endpoint and returns the result.

### 4️⃣ Slave Web Interface (`slave_gui.py` or equivalent)
- Built with Streamlit.
- Allows the user to:
  - Enter a database name.
  - Enter an SQL query (SELECT, INSERT, UPDATE, DELETE).
  - Execute it via the slave API.
  - View the returned results in a data table.
  - Handles decoding Base64-encoded results (for safety).

---

## 🛠️ Technologies Used

- **Go (Golang)** – for backend servers.
- **Streamlit (Python)** – for GUI interfaces.
- **MySQL** – as the relational database backend.
- **REST APIs** – for communication between components.

---

## 🚀 How to Run the System

1. **Start MySQL** on `localhost:3306` with:
   - Username: `root`
   - Password: `12345678`

2. **Start the Master Server**
```bash
go run master.go
```

3. **Start a Slave Server**
```bash
go run slave.go
```

4. **Launch the Master Web Interface**
```bash
streamlit run master.py
```

5. **Launch the Slave Web Interface**
```bash
streamlit run slave_gui.py
```

---

## 📄 Allowed SQL Commands

- ✅ Allowed from slaves: `SELECT`, `INSERT`, `UPDATE`, `DELETE`
- ❌ Not allowed from slaves: `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, etc.
- ✅ Full support for all queries from the master directly.

---

## 📋 Query Logging

All queries sent from slaves are logged in the `history_query.queries` table with:

- The executed query text
- Slave IP address
- Target database and table
- Timestamp of execution

---

## 👥 Use Cases

- Education in distributed systems and DB architectures.
- Lightweight SQL lab environment.
- Basic replication-style system simulation.

---

## 📌 Notes

- Make sure MySQL is running and accessible.
- Ensure Streamlit and requests libraries are installed in Python environment.

---

## 📧 Authors

Prepared by your team - Distributed DB Project, 2025

