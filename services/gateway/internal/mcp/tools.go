package mcp

// ToolDefinition describes an MCP tool conforming to Model Context Protocol specification.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema ToolInputSchema `json:"inputSchema"`
}

// ToolInputSchema defines the JSON Schema for tool arguments.
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef describes a property in JSON Schema.
type PropertyDef struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Items       map[string]string `json:"items,omitempty"`
	Default     any               `json:"default,omitempty"`
}

// ToolContent represents a content item in tool execution results.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolCallResult represents the result payload of a tool call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError"`
}

// DefaultTools returns the 3 MCP tools definitions according to project requirements.
func DefaultTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "errorium_search_solution",
			Description: "Поиск релевантных тем и кода решений по описанию ошибки или стектрейсу в Stack Overflow",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"query": {
						Type:        "string",
						Description: "Описание ошибки, стектрейс или поисковый запрос",
					},
					"tags": {
						Type:        "array",
						Description: "Теги технологий (например python, go)",
						Items: map[string]string{
							"type": "string",
						},
					},
					"limit": {
						Type:        "integer",
						Description: "Количество результатов (1-20, по умолчанию 5)",
						Default:     5,
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "errorium_explain_error",
			Description: "Получение детального RAG-разбора ошибки с анализом причин и рекомендациями",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"error_message": {
						Type:        "string",
						Description: "Текст сообщения об ошибке",
					},
					"stack_trace": {
						Type:        "string",
						Description: "Стектрейс ошибки (опционально)",
					},
				},
				Required: []string{"error_message"},
			},
		},
		{
			Name:        "errorium_get_related_errors",
			Description: "Запрос связанных тредов (дубликатов, ссылок) для выявленной проблемы",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"error_name": {
						Type:        "string",
						Description: "Название или идентификатор ошибки",
					},
				},
				Required: []string{"error_name"},
			},
		},
	}
}
