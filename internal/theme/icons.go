package theme

// Directory and tree icons
const (
	IconDirCollapsed = "▸"
	IconDirExpanded  = "▾"
	IconFile         = ""
	IconFileModified = "●"
	IconFileBinary   = ""
)

// Tree connector characters
const (
	TreeBranch     = "├── "
	TreeLastBranch = "└── "
	TreeVertical   = "│   "
	TreeSpace      = "    "
)

// Git status indicators
const (
	GitModified  = "[M]"
	GitAdded     = "[+]"
	GitDeleted   = "[D]"
	GitUntracked = "[?]"
	GitConflict  = "[!]"
	GitRenamed   = "[R]"
	GitCopied    = "[C]"
)

// Git branch icons
const (
	GitBranchIcon = ""
	GitAhead      = "↑"
	GitBehind     = "↓"
	GitDirty      = "*"
)

// Panel decorations
const (
	PanelDiamond = "◈"
)

// Status indicators
const (
	StatusRunning = "●"
	StatusIdle    = "○"
)

// Spinner frames for loading animations
var SpinnerDots = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var SpinnerPulse = []string{"░", "▒", "▓", "█", "▓", "▒"}

// FileIcons maps file extensions to Nerd Font icons
var FileIcons = map[string]string{
	// Go
	".go":  "󰟓",
	".mod": "󰏗",
	".sum": "󰏗",

	// Web
	".js":   "",
	".ts":   "",
	".tsx":  "",
	".jsx":  "",
	".html": "",
	".css":  "",
	".scss": "",
	".vue":  "",
	".svelte": "",

	// Data
	".json": "",
	".yaml": "",
	".yml":  "",
	".toml": "",
	".xml":  "",

	// Documentation
	".md":       "󰍔",
	".mdx":      "󰍔",
	".txt":      "",
	".rst":      "",

	// Config
	".env":        "󰈙",
	".gitignore":  "",
	".dockerignore": "",

	// Shell
	".sh":   "",
	".bash": "",
	".zsh":  "",
	".fish": "",

	// Python
	".py":  "",
	".pyi": "",
	".pyc": "",

	// Rust
	".rs": "",

	// C/C++
	".c":   "",
	".h":   "",
	".cpp": "",
	".hpp": "",

	// Java/Kotlin
	".java": "",
	".kt":   "",

	// Ruby
	".rb":   "",
	".rake": "",

	// Docker
	"Dockerfile": "",
	".dockerfile": "",

	// Git
	".git": "",

	// Images
	".png":  "",
	".jpg":  "",
	".jpeg": "",
	".gif":  "",
	".svg":  "",
	".ico":  "",

	// Archives
	".zip": "",
	".tar": "",
	".gz":  "",

	// Default
	"": "",
}

// DirIcons maps directory names to Nerd Font icons
var DirIcons = map[string]string{
	".git":         "\ue702", // Git icon
	"node_modules": "\ue718", // Node icon
	"vendor":       "\uf487", // Package icon
	"src":          "\uf07c", // Folder open
	"pkg":          "\uf487", // Package
	"cmd":          "\uf120", // Terminal
	"internal":     "\uf023", // Lock
	"test":         "\uf0c3", // Flask
	"tests":        "\uf0c3", // Flask
	"docs":         "\uf02d", // Book
	"doc":          "\uf02d", // Book
	"build":        "\uf085", // Cogs
	"dist":         "\uf49e", // Box
	"bin":          "\uf489", // Binary
	"lib":          "\uf02d", // Book
	"config":       "\uf013", // Cog
	".config":      "\uf013", // Cog
	".github":      "\uf408", // GitHub
	".vscode":      "\ue70c", // VS Code
	"api":          "\uf1c0", // Database/API
}

// GetFileIcon returns the appropriate icon for a file extension.
func GetFileIcon(ext string) string {
	if icon, ok := FileIcons[ext]; ok {
		return icon
	}
	return FileIcons[""]
}

// GetDirIcon returns the appropriate icon for a directory name.
func GetDirIcon(name string) string {
	if icon, ok := DirIcons[name]; ok {
		return icon
	}
	return ""
}
