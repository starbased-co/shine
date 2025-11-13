package help

import (
	"sort"
)

// Topic represents a help topic with metadata and content
type Topic struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Synopsis string   `json:"synopsis"`
	Usage    string   `json:"usage,omitempty"`
	Related  []string `json:"related,omitempty"`
	Content  string   `json:"-"` // Markdown content (exclude from JSON)
}

// Registry manages help topics
type Registry struct {
	topics map[string]*Topic
}

// NewRegistry creates a new help registry
func NewRegistry() *Registry {
	return &Registry{
		topics: make(map[string]*Topic),
	}
}

// Register adds a topic to the registry
func (r *Registry) Register(topic *Topic) {
	r.topics[topic.Name] = topic
}

// Get retrieves a topic by name
func (r *Registry) Get(name string) (*Topic, bool) {
	t, ok := r.topics[name]
	return t, ok
}

// List returns all topics sorted by name
func (r *Registry) List() []*Topic {
	topics := make([]*Topic, 0, len(r.topics))
	for _, t := range r.topics {
		topics = append(topics, t)
	}
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].Name < topics[j].Name
	})
	return topics
}

// ListByCategory returns topics grouped by category
func (r *Registry) ListByCategory() map[string][]*Topic {
	byCategory := make(map[string][]*Topic)
	for _, t := range r.topics {
		byCategory[t.Category] = append(byCategory[t.Category], t)
	}
	// Sort within each category
	for cat := range byCategory {
		topics := byCategory[cat]
		sort.Slice(topics, func(i, j int) bool {
			return topics[i].Name < topics[j].Name
		})
	}
	return byCategory
}

// Categories returns all unique categories sorted
func (r *Registry) Categories() []string {
	seen := make(map[string]bool)
	for _, t := range r.topics {
		seen[t.Category] = true
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// Names returns all topic names sorted
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.topics))
	for name := range r.topics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
