// Prism.js - Syntax highlighting for Go
// Placeholder - download full Prism.js from https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js
// And Prism Go language support from https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-go.min.js

// Simple syntax highlighting for Go (for demo purposes)
window.Prism = window.Prism || {};

// Minimal highlighting function
Prism.highlightElement = function(element) {
    // This is a placeholder - actual Prism.js should be loaded
    // For now, just return the element as-is
    // The code will be displayed with FiraCode font and ligatures
};

// Simple tokenizer for Go (placeholder)
Prism.languages = Prism.languages || {};
Prism.languages.go = {
    'comment': /\/\/.*|\/\*[\s\S]*?\*\//,
    'string': /(["'`])(?:(?!\1)[^\\]|\\[\s\S])*\1/,
    'keyword': /\b(?:break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b/,
    'function': /\b[a-zA-Z_][a-zA-Z0-9_]*\s*(?=\()/,
    'number': /\b\d+\.?\d*\b/,
    'operator': /[-+*/%&|^<>=!]|&&|\|\||<<|>>/,
    'punctuation': /[{}()[\],;.:]/
};
