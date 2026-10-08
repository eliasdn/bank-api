import re

with open('internal/handlers/users.go', 'r') as f:
    content = f.read()

count_not_found = content.count('"error": "user not found"')
count_db_error = content.count('"error": "database error"')

print(f"user not found: {count_not_found}")
print(f"database error: {count_db_error}")
