# JSON Server

JSON Server is a Go-based web server that dynamically serves HTML content based on JSON files. It supports custom templating, AI-driven design adjustments, and integration with popular CSS frameworks.

## Features

- **Dynamic Content:** Serves HTML based on `index.json` or `index.<name>.json` files.
- **Ordered JSON Parsing:** Preserves the order of keys in JSON objects for consistent rendering.
- **Custom Templating:** Supports `html/template` for rendering custom HTML components.
- **AI Design Mode:** (Optional) Generates basic styles based on a `designprompt` in your JSON, caching designs by UUID.
- **CSS Framework Integration:** Easily include Bootstrap, Tailwind CSS, Bulma, or Materialize via flags in your JSON.
- **Favicon Support:** Serves a `favicon.png` from the `assets` directory.
- **Static File Serving:** Serves static assets from the `assets` directory.

## Getting Started

### Prerequisites

- Go 1.16 or higher

### Installation

1. Clone the repository:
   ```bash
   git clone <repository_url>
   cd json-server
   ```

2. Create necessary directories:
   ```bash
   mkdir -p assets components components/cached
   ```

3. Create a `favicon.png` in the `assets` directory.

4. Create a default `index.json` file in the root directory (see "Usage" for example).

5. Create some default HTML templates in the `components` directory (see "Templating" for example).

### Running the Server

To run with default settings:

```bash
go run main.go
```

To enable AI design mode:

```bash
go run main.go -ai-design
```

The server will start on `http://localhost:8080`.

## Usage

### JSON Structure

The server reads `index.json` by default. You can specify other JSON files using the URL path, e.g., `/index.about` will load `index.about.json`.

Here's an example `index.json`:

```json
{
  "flags": {
    "csslib": "bootstrap",
    "designprompt": "dark moody"
  },
  "001": {
    "h1": "Welcome to My Page",
    "div": "This is some introductory content served by the JSON server.",
    "p": "You can define your content structure directly in JSON."
  },
  "002": {
    "h2": "Features",
    "ul": [
      "Dynamic Content",
      "Custom Templates",
      "AI Design Mode"
    ]
  },
  "003": {
    "img": "/assets/my-image.png"
  },
  "myCustomTag": {
    "message": "Hello from a custom tag!",
    "value": 123
  }
}
```

- **`flags`**: A special object for server-side configurations.
  - `csslib`: (Optional) Specifies a CSS framework (`bootstrap`, `tailwind`, `bulma`, `materialize`).
  - `designprompt`: (Optional, when `-ai-design` is enabled) A string used to generate dynamic styles.
- Numbered objects (`"001"`, `"002"`, etc.): These represent blocks of content that will be rendered in order. The keys within these objects are treated as HTML tags or template names.
- **Non-Standard Tags**: If a key within a content block is not a standard HTML tag (e.g., `myCustomTag` above) and does not have a corresponding template, its content will be injected into the HTML as a JavaScript variable (`customContent['myCustomTag']`).

### Templating

You can create HTML files in the `components` directory that match your JSON keys. For example, to handle the `"h1"` key in your JSON, create `components/h1.html`:

**`components/h1.html`:**

```html
<h1 style="color: blue;">{{.}}</h1>
```

This template will receive the value associated with the `h1` key in your JSON (e.g., "Welcome to My Page") and render it within the `<h1>` tags.

You can also create a default `div.html`:

**`components/div.html`:**

```html
<div style="border: 1px solid #ccc; padding: 10px;">
    {{.}}
</div>
```

If no template is found for a standard HTML tag (like `p`), the server will fall back to rendering it as a basic HTML tag (e.g., `<p>Content</p>`).

### AI Design Mode

When `ai-design` flag is enabled, the server will:

1. **Check for existing designs:** It first looks for a cached design based on the `designprompt` value in `components/cached/UUID/prompt.txt`.
2. **Generate new design:** If no cached design is found, it generates a new set of basic templates (`h1.html`, `div.html`) in a new UUID-named directory under `components/cached/`. The generation is based on keywords in the `designprompt` (e.g., "dark", "moody", "clean", "serif").
3. **Override templates:** These generated templates will override any default templates in the `components` directory.

### Assets

Place any static assets (images, CSS, JS) in the `assets` directory. They will be served from `/assets/`. For example, `assets/my-image.png` will be accessible at `http://localhost:8080/assets/my-image.png`.

## Code Structure

- `main.go`: Contains the main server logic, request handling, JSON parsing, templating, and AI design generation.
- `assets/`: Directory for static files (e.g., `favicon.png`).
- `components/`: Directory for default HTML templates.
- `components/cached/`: Directory for AI-generated design templates. Each subfolder is a UUID representing a cached design.

## Contributing

Feel free to open issues or submit pull requests.

### Todo

- [ ] implement llm features with api key
- [ ] remove exe flags 
