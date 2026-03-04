// Termynal - A lightweight, dependency-free JS library for animating terminal-like content
// Version: 0.2.0
// Author: Alex Lohr
// License: MIT

'use strict';

class Termynal {
    constructor(container, options = {}) {
        this.container = (typeof container === 'string') ? document.querySelector(container) : container;
        if (!this.container) {
            console.error('Termynal: Container not found');
            return;
        }

        this.options = Object.assign({
            startDelay: 100,
            typeDelay: 20,
            lineDelay: 50,
            cursorChar: '▋',
            cursorVisible: true,
            noDelay: false
        }, options);

        this.originalContent = this.container.innerHTML;
        this.lines = [];
        this.cursor = null;
        this.currentIndex = 0;
        this.currentLine = null;
        this.init();
    }

    init() {
        // Reset content
        this.container.innerHTML = '';

        // Parse original content
        this.lines = this.parseContent(this.originalContent);

        // Add cursor
        this.addCursor();

        // Start typing if not delayed
        if (!this.options.noDelay) {
            setTimeout(() => this.typeNextLine(), this.options.startDelay);
        } else {
            this.typeNextLine();
        }
    }

    parseContent(content) {
        const div = document.createElement('div');
        div.innerHTML = content;
        return Array.from(div.childNodes).map(node => node.cloneNode(true));
    }

    addCursor() {
        this.cursor = document.createElement('span');
        this.cursor.className = 'termynal-cursor';
        this.cursor.textContent = this.options.cursorChar;
        this.cursor.style.animation = 'blink 1s step-end infinite';
        this.container.appendChild(this.cursor);
    }

    removeCursor() {
        if (this.cursor && this.cursor.parentNode) {
            this.cursor.parentNode.removeChild(this.cursor);
        }
    }

    typeNextLine() {
        if (this.currentIndex >= this.lines.length) {
            this.removeCursor();
            this.container.dispatchEvent(new Event('termynal-complete'));
            console.log('Animation complete');
            return;
        }

        this.currentLine = this.lines[this.currentIndex];
        this.container.insertBefore(this.currentLine, this.cursor);
        this.currentIndex++;

        // Type the line content
        this.typeLine(this.currentLine).then(() => {
            setTimeout(() => this.typeNextLine(), this.options.lineDelay);
        });
    }

    typeLine(line) {
        return new Promise(resolve => {
            if (line.nodeType === Node.TEXT_NODE) {
                // Text node - type character by character
                this.typeText(line, resolve);
            } else if (line.nodeType === Node.ELEMENT_NODE) {
                // Element node - type recursively or reveal
                const codeElement = line.querySelector('code');
                if (codeElement) {
                    this.typeCode(codeElement, resolve);
                } else {
                    // Just reveal the element
                    resolve();
                }
            } else {
                resolve();
            }
        });
    }

    typeText(textNode, resolve) {
        const text = textNode.textContent;
        textNode.textContent = '';
        let index = 0;

        const typeChar = () => {
            if (index < text.length) {
                textNode.textContent += text[index];
                index++;
                setTimeout(typeChar, this.options.typeDelay);
            } else {
                resolve();
            }
        };

        typeChar();
    }

    typeCode(codeElement, resolve) {
        const text = codeElement.textContent;
        codeElement.textContent = '';
        let index = 0;

        const typeChar = () => {
            if (index < text.length) {
                // Preserve newlines and indentation
                if (text[index] === '\n') {
                    codeElement.textContent += '\n';
                } else {
                    codeElement.textContent += text[index];
                }
                index++;
                setTimeout(typeChar, this.options.typeDelay);
            } else {
                // Re-apply syntax highlighting
                if (window.Prism) {
                    Prism.highlightElement(codeElement);
                }
                resolve();
            }
        };

        typeChar();
    }

    getRemainingTime() {
        let remaining = 0;
        for (let i = this.currentIndex; i < this.lines.length; i++) {
            const line = this.lines[i];
            if (line.nodeType === Node.TEXT_NODE) {
                remaining += line.textContent.length * this.options.typeDelay;
            } else if (line.nodeType === Node.ELEMENT_NODE) {
                const codeElement = line.querySelector('code');
                if (codeElement) {
                    remaining += codeElement.textContent.length * this.options.typeDelay;
                }
            }
            remaining += this.options.lineDelay;
        }
        return remaining + this.options.startDelay;
    }
}

// Add CSS for cursor animation
const style = document.createElement('style');
style.textContent = `
@keyframes blink {
    0%, 50% { opacity: 1; }
    51%, 100% { opacity: 0; }
}
.termynal-cursor {
    color: #4ec9b0;
    font-weight: bold;
}`;
document.head.appendChild(style);

// Auto-initialize if data-termynal attribute is present
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-termynal]').forEach(container => {
        const noDelay = container.hasAttribute('data-termynal-no-delay');
        new Termynal(container, { noDelay });
    });
});
