package app

// PanelID identifies which panel has focus.
type PanelID int

const (
	PanelFileTree PanelID = iota
	PanelContent
	PanelMiniBuffer
)

// String returns the panel name for debugging.
func (p PanelID) String() string {
	switch p {
	case PanelFileTree:
		return "FileTree"
	case PanelContent:
		return "Content"
	case PanelMiniBuffer:
		return "MiniBuffer"
	default:
		return "Unknown"
	}
}

// FocusMsg requests focus change to a specific panel.
type FocusMsg struct {
	Target PanelID
}

// FocusCycleMsg cycles focus to the next or previous panel.
type FocusCycleMsg struct {
	Direction int // +1 forward, -1 backward
}

// ToggleMiniBufferMsg toggles the mini buffer visibility.
type ToggleMiniBufferMsg struct{}

// OpenFileMsg requests a file to be opened in the content pane.
type OpenFileMsg struct {
	Path string
}

// RunAIMsg requests launching an AI assistant in the terminal.
type RunAIMsg struct {
	Provider string // Provider name (empty = default)
}

// ContentMode determines what the content pane displays.
type ContentMode int

const (
	ModeViewer ContentMode = iota
	ModeDiff
	ModeTerminal
)

// ContentModeMsg changes the content pane mode.
type ContentModeMsg struct {
	Mode ContentMode
}

// FileLoadedMsg indicates a file has been loaded.
type FileLoadedMsg struct {
	Path    string
	Content []byte
	Err     error
}


// ErrorMsg represents an error that should be displayed.
type ErrorMsg struct {
	Err error
}

// StatusMsg updates the status bar with a message.
type StatusMsg struct {
	Text string
}
