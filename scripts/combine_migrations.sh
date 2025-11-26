#!/bin/bash
# combine_migrations.sh

# Ensure we are in the project root
cd "$(dirname "$0")/.."

echo "Combining migration files..."

for upfile in migrations/*.up.sql; do
    if [ ! -f "$upfile" ]; then continue; fi
    
    base=$(basename "$upfile" .up.sql)
    downfile="migrations/${base}.down.sql"
    newfile="migrations/${base}.sql"
    
    echo "Processing $base..."
    
    # Create new file with Up marker
    echo "-- +goose Up" > "$newfile"
    
    # Append Up content (skipping existing markers if any)
    grep -v "^-- +goose Up" "$upfile" >> "$newfile"
    
    # Add Down marker
    echo "" >> "$newfile"
    echo "-- +goose Down" >> "$newfile"
    
    # Append Down content if it exists
    if [ -f "$downfile" ]; then
        cat "$downfile" >> "$newfile"
        rm "$downfile"
    else
        echo "Warning: No down file for $base"
    fi
    
    # Remove old up file
    rm "$upfile"
done

echo "✅ Migrations combined successfully!"
