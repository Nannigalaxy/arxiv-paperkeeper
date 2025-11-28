#!/bin/bash

# ArXiv PaperKeeper - Manual Testing Script
# This script helps verify all functionality is working

echo "🧪 ArXiv PaperKeeper - Manual Testing Guide"
echo "========================================"
echo ""

# Check if server is running
echo "1. Checking if server is running..."
if curl -s http://localhost:8080 > /dev/null; then
    echo "   ✅ Server is running on port 8080"
else
    echo "   ❌ Server is not running. Start with: ./bin/arxiv-paperkeeper server"
    exit 1
fi

echo ""
echo "2. Testing Homepage..."
HOMEPAGE=$(curl -s http://localhost:8080)
if echo "$HOMEPAGE" | grep -q "Browse Papers"; then
    echo "   ✅ Homepage loads correctly"
else
    echo "   ❌ Homepage not loading properly"
fi

echo ""
echo "3. Testing Paper Count..."
PAPER_COUNT=$(echo "SELECT COUNT(*) FROM papers;" | sqlite3 data/arxiv.db 2>/dev/null || echo "0")
echo "   📊 Papers in database: $PAPER_COUNT"

echo ""
echo "4. Testing Library Count..."
LIBRARY_COUNT=$(echo "SELECT COUNT(*) FROM library;" | sqlite3 data/arxiv.db 2>/dev/null || echo "0")
echo "   📚 Papers in library: $LIBRARY_COUNT"

echo ""
echo "5. Testing Tags..."
TAG_COUNT=$(echo "SELECT COUNT(*) FROM tags;" | sqlite3 data/arxiv.db 2>/dev/null || echo "0")
echo "   🏷️  Tags created: $TAG_COUNT"

echo ""
echo "📝 Manual Testing Checklist:"
echo "   Open http://localhost:8080 in your browser and test:"
echo ""
echo "   [ ] Homepage displays papers"
echo "   [ ] Search works (try searching for 'learning')"
echo "   [ ] Click on a paper to view details"
echo "   [ ] 'Save to Library' button works"
echo "   [ ] Navigate to 'My Library' page"
echo "   [ ] 'Mark as Read' button works"
echo "   [ ] Add a tag to a paper"
echo "   [ ] Remove a tag from a paper"
echo "   [ ] Remove paper from library"
echo "   [ ] Pagination works (Next/Previous)"
echo "   [ ] PDF and arXiv links work"
echo ""
echo "✨ Happy testing!"
