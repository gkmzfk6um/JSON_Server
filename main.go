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

// OrderedPair represents a key-value pair with preserved order
type OrderedPair struct {
	Key   string
	Value interface{}
}

// ContentItem represents the content inside the ID object with preserved order
type ContentItem struct {
	ID      string
	Content []OrderedPair
}

var aiDesign bool
var templates *template.Template

func serveFavicon(w http.ResponseWriter, r *http.Request) {
	//adjust content type if you use .ico instead
	w.Header().Set("Content-Type", "image/png")

	data, err := ioutil.ReadFile("assets/favicon.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func main() {
	flag.BoolVar(&aiDesign, "ai-design", false, "Enable AI design mode for enhanced styling")
	flag.Parse()

	// Initial template parsing (default)
	parseTemplates("")

	http.HandleFunc("/favicon.ico", serveFavicon)
	http.Handle("/assets/", http.StripPrefix("/assets/",
		http.FileServer(http.Dir("assets")),
	))

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
	// Determine which JSON file to load
	jsonFile := "index.json"

	// Check if the path is /index.something
	if r.URL.Path != "/" {
		if strings.HasPrefix(r.URL.Path, "/index.") {
			// Extract the name after /index.
			name := strings.TrimPrefix(r.URL.Path, "/index.")
			if name != "" {
				jsonFile = "index." + name + ".json"
			}
		} else {
			http.NotFound(w, r)
			return
		}
	}

	data, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not read %s", jsonFile), http.StatusInternalServerError)
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

	// Parse JSON to extract key order
	contentItems, err := parseOrderedJSON(data)
	if err != nil {
		http.Error(w, "Could not parse JSON with order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	renderHTML(w, contentItems, flags)
}

// parseOrderedJSON parses JSON while preserving the order of keys
func parseOrderedJSON(data []byte) ([]ContentItem, error) {
	// First, parse normally to get the data
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// Extract the order of keys from the raw JSON
	keyOrder := extractJSONKeyOrder(string(data))

	// Build content items in order
	var contentItems []ContentItem
	for _, topKey := range keyOrder {
		if topKey == "flags" {
			continue
		}

		if contentMap, ok := jsonData[topKey].(map[string]interface{}); ok {
			// Get the order of keys within this content item
			innerKeyOrder := extractInnerKeyOrder(string(data), topKey)

			var pairs []OrderedPair
			for _, innerKey := range innerKeyOrder {
				if value, exists := contentMap[innerKey]; exists {
					pairs = append(pairs, OrderedPair{Key: innerKey, Value: value})
				}
			}
			contentItems = append(contentItems, ContentItem{
				ID:      topKey,
				Content: pairs,
			})
		}
	}

	return contentItems, nil
}

// extractJSONKeyOrder extracts the order of top-level keys from raw JSON
func extractJSONKeyOrder(jsonStr string) []string {
	var keys []string
	decoder := json.NewDecoder(strings.NewReader(jsonStr))

	// Read opening brace
	decoder.Token()

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if key, ok := token.(string); ok {
			keys = append(keys, key)
			// Skip the value
			var dummy interface{}
			decoder.Decode(&dummy)
		}
	}

	return keys
}

// extractInnerKeyOrder extracts the order of keys within a specific object
func extractInnerKeyOrder(jsonStr string, objectKey string) []string {
	var keys []string

	// Find the object in the JSON string
	// This is a simplified approach - look for "objectKey": {
	searchStr := fmt.Sprintf("\"%s\":", objectKey)
	idx := strings.Index(jsonStr, searchStr)
	if idx == -1 {
		return keys
	}

	// Find the opening brace after the key
	startIdx := strings.Index(jsonStr[idx:], "{")
	if startIdx == -1 {
		return keys
	}
	startIdx += idx + 1

	// Extract the substring for this object
	braceCount := 1
	endIdx := startIdx
	for endIdx < len(jsonStr) && braceCount > 0 {
		if jsonStr[endIdx] == '{' {
			braceCount++
		} else if jsonStr[endIdx] == '}' {
			braceCount--
		}
		endIdx++
	}

	objectStr := jsonStr[startIdx : endIdx-1]

	// Parse the keys from this substring
	decoder := json.NewDecoder(strings.NewReader("{" + objectStr + "}"))
	decoder.Token() // Read opening brace

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if key, ok := token.(string); ok {
			keys = append(keys, key)
			// Skip the value
			var dummy interface{}
			decoder.Decode(&dummy)
		}
	}

	return keys
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

	// H1 Template
	h1Content := fmt.Sprintf(`<h1 style="color: %s; font-family: %s; border-bottom: 2px solid %s;">{{.}}</h1>`, accentColor, font, accentColor)
	ioutil.WriteFile(filepath.Join(dir, "h1.html"), []byte(h1Content), 0644)

	// Div Template
	divContent := fmt.Sprintf(`<div style="background: %s; color: %s; padding: 20px; border-radius: 8px; margin: 10px 0;">{{.}}</div>`, bgColor, textColor)
	ioutil.WriteFile(filepath.Join(dir, "div.html"), []byte(divContent), 0644)
}

func generateUUID() string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	return hex.EncodeToString(h.Sum(nil))
}

func renderHTML(w http.ResponseWriter, items []ContentItem, flags map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlStart := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JSON Server</title>
`

	// Add CSS library if specified in flags
	if cssLib, ok := flags["csslib"]; ok && cssLib != nil {
		cssLibStr := fmt.Sprintf("%v", cssLib)
		switch strings.ToLower(cssLibStr) {
		case "bootstrap":
			htmlStart += `    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
`
		case "tailwind":
			htmlStart += `    <script src="https://cdn.tailwindcss.com"></script>
`
		case "bulma":
			htmlStart += `    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css">
`
		case "materialize":
			htmlStart += `    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/materialize/1.0.0/css/materialize.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/materialize/1.0.0/js/materialize.min.js"></script>
`
		}
	}

	htmlStart += `    <style>
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
		for _, pair := range item.Content {
			tag := pair.Key
			content := pair.Value

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
		htmlStart += ``
	}

	htmlStart += `</head><body><div class="container">`
	fmt.Fprint(w, htmlStart)

	for _, item := range items {
		// Wrap each numbered object in a div
		fmt.Fprintf(w, "<div id='%s'>", item.ID)

		for _, pair := range item.Content {
			tag := pair.Key
			content := pair.Value

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

		// Close the div wrapper
		fmt.Fprint(w, "</div>")
	}

	fmt.Fprint(w, `</div></body></html>`)
}
