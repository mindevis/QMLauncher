#!/bin/bash

# Script to validate conventional commits locally

echo "🔍 Validating conventional commits..."
echo

# Check if commitlint is available
if command -v commitlint >/dev/null 2>&1; then
    echo "📋 Checking last commit with commitlint..."
    if git log -1 --pretty=format:"%s" | npx commitlint --verbose; then
        echo "✅ Last commit follows conventional commit format"
    else
        echo "❌ Last commit does not follow conventional commit format"
    fi
else
    echo "⚠️ commitlint not found, performing basic validation..."
    # Basic validation without commitlint
    LAST_COMMIT=$(git log -1 --pretty=format:"%s")
    echo "📋 Last commit: $LAST_COMMIT"

    # Check basic format: type: description
    if echo "$LAST_COMMIT" | grep -E '^(feat|fix|docs|style|refactor|test|chore|ci|build|perf)(\(.+\))?: .+' >/dev/null; then
        echo "✅ Last commit follows basic conventional commit format"
    else
        echo "❌ Last commit does not follow conventional commit format"
        echo
        echo "📝 Conventional commit format:"
        echo "type(scope): description"
        echo
        echo "Types: feat, fix, docs, style, refactor, test, chore, ci, build, perf"
        echo
        echo "Examples:"
        echo "feat: add new feature"
        echo "fix: resolve bug in launcher"
        echo "docs: update README"
        echo "chore: update dependencies"
        echo
        echo "💡 Install commitlint for full validation:"
        echo "npm install -g @commitlint/cli @commitlint/config-conventional"
        exit 1
    fi
fi

echo
echo "🎉 All commits are valid!"