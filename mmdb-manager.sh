#!/bin/bash

# GeoLite2-City.mmdb chunk/reassemble manager
# Usage: ./mmdb-manager.sh [chunk|reassemble]

set -e

MMDB_FILE="GeoLite2-City.mmdb"
CHUNK_SIZE="10M"
CHUNK_PREFIX="GeoLite2-City.mmdb.chunk."

chunk_mmdb() {
    if [ ! -f "$MMDB_FILE" ]; then
        echo "Error: $MMDB_FILE not found"
        exit 1
    fi

    echo "Splitting $MMDB_FILE into $CHUNK_SIZE chunks..."

    # Remove existing chunks
    rm -f ${CHUNK_PREFIX}*

    # Split the file into chunks
    split -b $CHUNK_SIZE "$MMDB_FILE" "$CHUNK_PREFIX"

    echo "Compressing chunks..."
    for chunk in ${CHUNK_PREFIX}*; do
        echo "  Compressing $chunk..."
        gzip "$chunk"
    done

    echo "Chunking complete!"
    echo "Created compressed chunks:"
    ls -lh ${CHUNK_PREFIX}*.gz
    echo ""
    echo "Total compressed size: $(du -ch ${CHUNK_PREFIX}*.gz | tail -1 | cut -f1)"
}

reassemble_mmdb() {
    echo "Reassembling $MMDB_FILE from chunks..."

    # Check if file already exists
    if [ -f "$MMDB_FILE" ]; then
        echo "$MMDB_FILE already exists, skipping reassembly"
        return 0
    fi

    # Check if chunks exist
    if ! ls ${CHUNK_PREFIX}*.gz 1> /dev/null 2>&1; then
        echo "Error: No chunks found matching ${CHUNK_PREFIX}*.gz"
        exit 1
    fi

    # Decompress and concatenate chunks in order
    for chunk_gz in ${CHUNK_PREFIX}*.gz; do
        echo "  Processing $chunk_gz..."
        gunzip -c "$chunk_gz" >> "$MMDB_FILE"
    done

    echo "Reassembly complete: $MMDB_FILE"
    echo "File size: $(ls -lh $MMDB_FILE | awk '{print $5}')"
}

case "${1:-}" in
    "chunk")
        chunk_mmdb
        ;;
    "reassemble")
        reassemble_mmdb
        ;;
    *)
        echo "Usage: $0 [chunk|reassemble]"
        echo ""
        echo "  chunk     - Split and compress $MMDB_FILE into git-friendly chunks"
        echo "  reassemble - Decompress and reassemble $MMDB_FILE from chunks"
        exit 1
        ;;
esac