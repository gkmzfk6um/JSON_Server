package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ContentItem represents the content inside the ID object
type ContentItem map[string]interface{}

var aiDesign bool
var templates *template.Template

func main() {
	flag.BoolVar(&aiDesign, "ai-design", false, "Enable AI design mode for enhanced styling")
	flag.Parse()

	// Initial template parsing (default)
	parseTemplates("")

	http.HandleFunc("/", handler)

	fmt.Println("Server starting on http://localhost:8080")
	if aiDesign {
		fmt.Println("AI Design Mode: ENABLED")
	}
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseTemplates(customUUID string) {
	var err error
	// Always load default templates first
	templates, err = template.ParseGlob("components/*.html")
	if err != nil {
		// It's okay if no default templates exist, but we should log it if it's an error other than no match
		if !strings.Contains(err.Error(), "pattern matches no files") {
			fmt.Println("Error parsing default templates:", err)
		}
	}

	// If a custom design is selected, load those templates on top (overriding defaults)
	if customUUID != "" {
		customPath := filepath.Join("components", "cached", customUUID, "*.html")
		customTemplates, err := template.ParseGlob(customPath)
		if err == nil {
			// If we already have templates, we need to merge or replace.
			// template.ParseGlob returns a *new* set.
			// To override, we can just use the custom set, assuming it might contain all needed overrides.
			// However, to support partial overrides, we should ideally parse into the existing set.
			// But ParseGlob creates a new one.
			// Strategy: Parse defaults, then parse custom into the SAME template instance?
			// template.Must(templates.ParseGlob(customPath)) would work if templates is not nil.
			if templates == nil {
				templates = customTemplates
			} else {
				_, err = templates.ParseGlob(customPath)
				if err != nil {
					fmt.Println("Error merging custom templates:", err)
				}
			}
		} else {
			fmt.Println("Error parsing custom templates:", err)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := ioutil.ReadFile("index.json")
	if err != nil {
		http.Error(w, "Could not read index.json", http.StatusInternalServerError)
		return
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		http.Error(w, "Could not parse index.json: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract flags (server-only)
	var flags map[string]interface{}
	var designPromptValue string
	var designUUID string

	if flagsData, ok := jsonData["flags"]; ok {
		if flagsMap, ok := flagsData.(map[string]interface{}); ok {
			flags = flagsMap
			// Check for designprompt in flags
			if prompt, ok := flags["designprompt"]; ok {
				designPromptValue = fmt.Sprintf("%v", prompt)
				if aiDesign {
					designUUID = getOrGenerateDesign(designPromptValue)
					// Re-parse templates with the new design
					parseTemplates(designUUID)
				}
			}
		}
	}

	// Extract content items (everything except "flags")
	contentItems := make([]map[string]interface{}, 0)
	for key, value := range jsonData {
		if key == "flags" {
			continue
		}
		if contentMap, ok := value.(map[string]interface{}); ok {
			contentItems = append(contentItems, contentMap)
		}
	}

	renderHTML(w, contentItems, flags)
}

func getOrGenerateDesign(prompt string) string {
	// 1. Check if prompt is a UUID (simple heuristic: length 32 hex)
	// If it looks like a UUID and exists in cached, return it.
	if len(prompt) == 32 {
		if _, err := os.Stat(filepath.Join("components", "cached", prompt)); err == nil {
			return prompt
		}
	}

	// 2. Check if we already have a generated design for this prompt
	// We can hash the prompt to find a consistent folder, or search.
	// Searching is safer if we want to avoid collisions or support manual UUIDs.
	// For simplicity, let's search all folders in components/cached for a matching prompt.txt
	cachedDir := filepath.Join("components", "cached")
	files, _ := ioutil.ReadDir(cachedDir)
	for _, f := range files {
		if f.IsDir() {
			promptPath := filepath.Join(cachedDir, f.Name(), "prompt.txt")
			content, err := ioutil.ReadFile(promptPath)
			if err == nil && strings.TrimSpace(string(content)) == strings.TrimSpace(prompt) {
				return f.Name()
			}
		}
	}

	// 3. Generate new design
	newUUID := generateUUID()
	newDir := filepath.Join(cachedDir, newUUID)
	if err := os.MkdirAll(newDir, 0755); err != nil {
		fmt.Println("Error creating cache dir:", err)
		return ""
	}

	// Save prompt
	ioutil.WriteFile(filepath.Join(newDir, "prompt.txt"), []byte(prompt), 0644)

	// Generate templates based on keywords
	generateTemplates(newDir, prompt)

	return newUUID
}

func generateTemplates(dir, prompt string) {
	promptLower := strings.ToLower(prompt)

	// Default styles
	bgColor := "#ffffff"
	textColor := "#333333"
	accentColor := "#3498db"
	font := "sans-serif"

	if strings.Contains(promptLower, "dark") {
		bgColor = "#2c3e50"
		textColor = "#ecf0f1"
		accentColor = "#e74c3c"
	}
	if strings.Contains(promptLower, "moody") {
		bgColor = "#1a1a1a"
		textColor = "#dcdcdc"
		accentColor = "#8e44ad"
	}
	if strings.Contains(promptLower, "clean") {
		font = "'Helvetica Neue', Helvetica, Arial, sans-serif"
	}
	if strings.Contains(promptLower, "serif") {
		font = "Georgia, serif"
	}

	// Generate a custom style block (we'll inject this via a special template or just rely on main layout if we had one)
	// Since we don't have a master layout file in the current setup (it's hardcoded in renderHTML),
	// we might need to generate a 'layout.html' if we refactored, but for now let's generate component overrides.

	// Actually, the current renderHTML hardcodes the HTML structure.
	// To support "AI Design", we should probably allow overriding the "structure" or at least the CSS.
	// The user said: "generate custom versiones of the default elements".

	// Let's generate a 'style.html' that we can include if we change renderHTML to look for it,
	// OR we can just generate component templates like 'h1.html', 'div.html' with inline styles or classes.

	// For this implementation, let's generate a 'css_override.html' and 'h1.html' as examples.

	// H1 Template
	h1Content := fmt.Sprintf(`<h1 style="color: %s; font-family: %s; border-bottom: 2px solid %s;">{{.}}</h1>`, accentColor, font, accentColor)
	ioutil.WriteFile(filepath.Join(dir, "h1.html"), []byte(h1Content), 0644)

	// Div Template
	divContent := fmt.Sprintf(`<div style="background: %s; color: %s; padding: 20px; border-radius: 8px; margin: 10px 0;">{{.}}</div>`, bgColor, textColor)
	ioutil.WriteFile(filepath.Join(dir, "div.html"), []byte(divContent), 0644)

	// We can also generate a special "style" template that renderHTML can check for?
	// Or better, let's just generate these component overrides for now as requested.
}

func generateUUID() string {
	// b := make([]byte, 16)

	// In a real app use crypto/rand, here simple is fine or just use md5 of time
	// For simplicity let's use a pseudo-random approach or just a simple unique string
	// Using MD5 of current time + randomish
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	return hex.EncodeToString(h.Sum(nil))
}

func renderHTML(w http.ResponseWriter, items []map[string]interface{}, flags map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlStart := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JSON Server</title>
    <style>
        body { font-family: sans-serif; line-height: 1.6; padding: 20px; max-width: 800px; margin: 0 auto; }
        img { max-width: 100%; height: auto; }
    </style>
`

	// Collect non-standard tags (tags without templates and not standard HTML)
	standardTags := map[string]bool{
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"p": true, "div": true, "span": true, "ul": true, "ol": true, "li": true,
		"img": true, "a": true, "button": true, "input": true, "form": true,
		"table": true, "tr": true, "td": true, "th": true, "thead": true, "tbody": true,
		"section": true, "article": true, "header": true, "footer": true, "nav": true,
		"main": true, "aside": true, "figure": true, "figcaption": true,
	}

	nonStandardData := make(map[string]interface{})

	// First pass: collect non-standard tags
	for _, item := range items {
		for tag, content := range item {
			// Check if it's a standard tag or has a template
			hasTemplate := false
			if templates != nil {
				if templates.Lookup(tag+".html") != nil || templates.Lookup(tag) != nil {
					hasTemplate = true
				}
			}

			if !standardTags[tag] && !hasTemplate {
				// Store in nonStandardData for JS injection
				nonStandardData[tag] = content
			}
		}
	}

	// Inject non-standard data as JavaScript variables
	if len(nonStandardData) > 0 {
		htmlStart += `<script>
        // Non-standard tag content accessible to client
        var customContent = {};
`
		for tag, content := range nonStandardData {
			jsonContent, _ := json.Marshal(content)
			htmlStart += fmt.Sprintf("        customContent['%s'] = %s;\n", tag, string(jsonContent))
		}
		htmlStart += `    </script>
`
	}

	if aiDesign {
		htmlStart += `
    <style>
        /* AI Design Mode Base Styles */
        body {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            min-height: 100vh;
        }
        .container {
            background: rgba(255,255,255,0.8);
            padding: 40px;
            border-radius: 12px;
            margin-top: 40px;
        }
    </style>
`
	}

	htmlStart += `</head><body><div class="container">`
	fmt.Fprint(w, htmlStart)

	for _, item := range items {
		for tag, content := range item {
			// Check if a template exists for this tag
			if templates != nil {
				if tmpl := templates.Lookup(tag + ".html"); tmpl != nil {
					if err := tmpl.Execute(w, content); err != nil {
						fmt.Fprintf(w, "<!-- Error rendering template %s: %v -->", tag, err)
					}
					continue
				}
				if tmpl := templates.Lookup(tag); tmpl != nil {
					if err := tmpl.Execute(w, content); err != nil {
						fmt.Fprintf(w, "<!-- Error rendering template %s: %v -->", tag, err)
					}
					continue
				}
			}

			// If it's a non-standard tag without a template, skip rendering (already in JS)
			if !standardTags[tag] {
				continue
			}

			switch tag {
			case "img":
				val := fmt.Sprintf("%v", content)
				fmt.Fprintf(w, `<img src="%s" alt="Image">`, val)
			case "ul":
				// Handle list items
				fmt.Fprint(w, "<ul>")
				if list, ok := content.([]interface{}); ok {
					for _, li := range list {
						fmt.Fprintf(w, "<li>%v</li>", li)
					}
				} else {
					// Fallback if it's not a list
					fmt.Fprintf(w, "<li>%v</li>", content)
				}
				fmt.Fprint(w, "</ul>")
			default:
				val := fmt.Sprintf("%v", content)
				fmt.Fprintf(w, `<%s>%s</%s>`, tag, val, tag)
			}
		}
	}

	fmt.Fprint(w, `</div></body></html>`)
}
