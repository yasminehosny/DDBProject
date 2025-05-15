import streamlit as st
import requests
import json
import pandas as pd
import base64

st.title("Master-Slave Database Query Interface")

# Inputs
dbname = st.text_input("Enter Database Name")
query = st.text_area("Enter SQL Query (e.g., SELECT * FROM city)")

# Button
if st.button("Execute Query"):
    if dbname and query:
        payload = {
            "dbname": dbname,
            "query": query
        }

        url = "http://localhost:8001/query"
        headers = {'Content-Type': 'application/json'}

        try:
            response = requests.post(url, data=json.dumps(payload), headers=headers)
            if response.status_code == 200:
                try:
                    results = response.json()
                except Exception:
                    st.write(response.text)
                    st.stop()

                if results.get('results'):
                    st.write("Query Results:")

                    # فك ترميز أي بيانات base64
                    for row in results['results']:
                        for key, value in row.items():
                            if isinstance(value, str):
                                try:
                                    decoded = base64.b64decode(value).decode('utf-8')
                                    row[key] = decoded
                                except Exception:
                                    pass

                    df = pd.DataFrame(results['results'])
                    st.dataframe(df)
                else:
                    st.info("No results returned.")
            else:
                st.error(f"Error: {response.text}")
        except Exception as e:
            st.error(f"Failed to connect to Slave API: {e}")
    else:
        st.warning("Please enter both database name and query.")
