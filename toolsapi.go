package unifai

import (
	"time"

	"github.com/unifai-network/unifai-sdk-go/common"
)

// ToolsAPI extends the common API for additional tool-related endpoints.
type ToolsAPI struct {
	*common.API
}

// NewToolsAPI creates a new ToolsAPI instance using the provided config.
// If no endpoint is provided in the config, it defaults to BACKEND_API_ENDPOINT.
func NewToolsAPI(config common.APIConfig) *ToolsAPI {
	if config.Endpoint == "" {
		config.Endpoint = common.BACKEND_API_ENDPOINT
	}
	return &ToolsAPI{
		API: common.NewAPI(config),
	}
}

// SearchTools sends a GET request to '/actions/search' with the given query parameters.
func (t *ToolsAPI) SearchTools(args map[string]string) (interface{}, error) {
	return t.Request("GET", "/actions/search", common.RequestOptions{
		Params: args,
	})
}

// CallTool sends a POST request to '/actions/call' with the provided JSON body and a 50 second timeout.
func (t *ToolsAPI) CallTool(args interface{}) (interface{}, error) {
	return t.Request("POST", "/actions/call", common.RequestOptions{
		JSON:    args,
		Timeout: 50 * time.Second,
	})
}
