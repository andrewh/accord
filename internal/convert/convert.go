// Orchestrate conversion from Pact contract files to Accord YAML.
package convert

// Warning represents a non-fatal issue found during conversion.
type Warning struct {
	File    string
	Message string
}
