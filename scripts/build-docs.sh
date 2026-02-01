#!/usr/bin/env bash
set -euo pipefail

# Change to docs directory where package.json lives
cd docs

echo "🔨 Building Wave documentation..."

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Build documentation
echo "📚 Building VitePress site..."
npm run build
npm run build

# Validate build
if [ -d "docs/.vitepress/dist" ]; then
    echo "✅ Documentation built successfully"
    echo "📂 Output: docs/.vitepress/dist/"
    
    # Count generated files
    file_count=$(find docs/.vitepress/dist -type f | wc -l)
    echo "📊 Generated $file_count files"
else
    echo "❌ Documentation build failed"
    exit 1
fi

# Check for broken internal links
echo "🔍 Checking internal links..."
docs_dir="docs"
broken_links=()

# Find all markdown links and check if target exists
while IFS= read -r -d '' line; do
    # Match markdown links [text](path)
    if [[ $line =~ \[.*\]\(([^)]+)\) ]]; then
        link="${BASH_REMATCH[1]}"
        # Skip external links
        if [[ $link != http* ]] && [[ $link != https* ]]; then
            # Remove hash fragments
            link_path="${link%%#*}"
            # Convert relative paths
            if [[ $link_path == /* ]]; then
                target_file="docs${link_path}"
            else
                target_file="docs/$(dirname "${BASH_SOURCE[1]}")/$link_path"
            fi
            
            if [ ! -f "$target_file" ] && [ ! -d "$target_file" ]; then
                broken_links+=("$link_path")
            fi
        fi
    fi
done < <(grep -Roh "\[.*\](" "$docs_dir" --include="*.md")

if [ ${#broken_links[@]} -gt 0 ]; then
    echo "⚠️  Found broken links:"
    printf '  %s\n' "${broken_links[@]}"
    exit 1
else
    echo "✅ All internal links resolve correctly"
fi

echo "🎉 Documentation build complete!"