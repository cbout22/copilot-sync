package manifest

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const DefaultManifestFile = "copilot.toml"

// Manifest represents the full copilot.toml file.
// Each section maps asset names to their remote references.
type Manifest struct {
	Instructions map[string]string `toml:"instructions,omitempty"`
	Agents       map[string]string `toml:"agents,omitempty"`
	Prompts      map[string]string `toml:"prompts,omitempty"`
	Skills       map[string]string `toml:"skills,omitempty"`
}

// New returns an empty Manifest with initialised maps.
func New() *Manifest {
	return &Manifest{
		Instructions: make(map[string]string),
		Agents:       make(map[string]string),
		Prompts:      make(map[string]string),
		Skills:       make(map[string]string),
	}
}

// Load reads and parses a copilot.toml file from the given path.
// If the file does not exist it returns an empty manifest (no error).
func Load(path string) (*Manifest, error) {
	m := New()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	if err := toml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	// Ensure nil maps are initialised
	if m.Instructions == nil {
		m.Instructions = make(map[string]string)
	}
	if m.Agents == nil {
		m.Agents = make(map[string]string)
	}
	if m.Prompts == nil {
		m.Prompts = make(map[string]string)
	}
	if m.Skills == nil {
		m.Skills = make(map[string]string)
	}

	return m, nil
}

// Save writes the manifest back to the given path.
func (m *Manifest) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating manifest file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(m); err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	return nil
}

// Section returns the map for the given asset type name.
func (m *Manifest) Section(assetType string) (map[string]string, error) {
	switch assetType {
	case "instructions":
		return m.Instructions, nil
	case "agents":
		return m.Agents, nil
	case "prompts":
		return m.Prompts, nil
	case "skills":
		return m.Skills, nil
	default:
		return nil, fmt.Errorf("unknown asset type: %s", assetType)
	}
}

// Set adds or updates an entry in the given asset type section.
func (m *Manifest) Set(assetType, name, ref string) error {
	section, err := m.Section(assetType)
	if err != nil {
		return err
	}
	section[name] = ref
	return nil
}

// Remove deletes an entry from the given asset type section.
// Returns true if the entry existed, false otherwise.
func (m *Manifest) Remove(assetType, name string) (bool, error) {
	section, err := m.Section(assetType)
	if err != nil {
		return false, err
	}
	if _, ok := section[name]; !ok {
		return false, nil
	}
	delete(section, name)
	return true, nil
}

// AllEntries returns every (type, name, ref) triple in the manifest.
func (m *Manifest) AllEntries() []Entry {
	var entries []Entry
	for name, ref := range m.Instructions {
		entries = append(entries, Entry{Type: "instructions", Name: name, Ref: ref})
	}
	for name, ref := range m.Agents {
		entries = append(entries, Entry{Type: "agents", Name: name, Ref: ref})
	}
	for name, ref := range m.Prompts {
		entries = append(entries, Entry{Type: "prompts", Name: name, Ref: ref})
	}
	for name, ref := range m.Skills {
		entries = append(entries, Entry{Type: "skills", Name: name, Ref: ref})
	}
	return entries
}

// Entry is a flattened manifest row.
type Entry struct {
	Type string
	Name string
	Ref  string
}
