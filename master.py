import streamlit as st
import requests
import pandas as pd

st.title("🧠 Master Server GUI")

# إنشاء/حذف قاعدة البيانات
st.header("1️⃣ Create/Delete Database")
db_name = st.text_input("Database Name")
if st.button("Create Database"):
    res = requests.post("http://localhost:8000/create_database", json={"dbname": db_name})
    st.success(res.text)
if st.button("Drop Database"):
    res = requests.post("http://localhost:8000/drop_database", json={"dbname": db_name})
    st.success(res.text)

# إنشاء/حذف الجدول
st.header("2️⃣ Create/Delete Table")
table_db = st.text_input("Database to Use for Table")
table_name = st.text_input("Table Name")
columns = st.text_area("Columns (e.g., id INT PRIMARY KEY, name VARCHAR(100))")
if st.button("Create Table"):
    res = requests.post("http://localhost:8000/create_table", json={"dbname": table_db, "table": table_name, "columns": columns.split(',')})
    st.success(res.text)
if st.button("Drop Table"):
    res = requests.post("http://localhost:8000/drop_table", json={"dbname": table_db, "table": table_name})
    st.success(res.text)

# عرض سجل الاستعلامات المحفوظة
st.header("📜 Query Logs")
if st.button("Load Query Logs"):
    try:
        res = requests.get("http://localhost:8000/queries_log")
        res.raise_for_status()
        logs = res.json()
        if logs:
            df = pd.DataFrame(logs)
            st.dataframe(df)
        else:
            st.info("No query logs found.")
    except Exception as e:
        st.error(f"Failed to load logs: {e}")
