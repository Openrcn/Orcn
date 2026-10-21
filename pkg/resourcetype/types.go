package resourcetype

import (
	"fmt"
	"regexp"
	"strings"
)

type ProviderField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Placeholder string `json:"placeholder,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	SpecKey     string `json:"spec_key"`
}

type Provider struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Fields      []ProviderField `json:"fields"`
	Validator   func(map[string]any) error `json:"-"`
}

var Providers = []Provider{
	{
		ID:          "http",
		Name:        "HTTP Direct URL",
		Description: "Download a file or archive from a direct HTTP(S) URL.",
		Fields: []ProviderField{
			{
				Name:        "url",
				Label:       "Resource URL",
				Type:        "text",
				Placeholder: "https://example.com/model.bin",
				Description: "Direct HTTP(S) link to the file or archive.",
				Required:    true,
				SpecKey:     "url",
			},
			{
				Name:        "target",
				Label:       "Target Mount Path",
				Type:        "text",
				Placeholder: "e.g. /workspace/models/checkpoints/",
				Description: "Absolute path inside the container where the resource is placed.",
				Required:    true,
				SpecKey:     "target",
			},
		},
	},
	{
		ID:          "hf",
		Name:        "Hugging Face",
		Description: "Download models, datasets or LoRAs from a Hugging Face repository.",
		Fields: []ProviderField{
			{
				Name:        "url",
				Label:       "Repository",
				Type:        "text",
				Placeholder: "org/repo (e.g. runwayml/stable-diffusion-v1-5)",
				Description: "Hugging Face repository ID in 'org/repo' format.",
				Required:    true,
				SpecKey:     "url",
			},
			{
				Name:        "files",
				Label:       "Files",
				Type:        "string_list",
				Placeholder: "e.g. model.safetensors",
				Description: "Specific files to download. Leave empty to pull the entire repository.",
				Required:    false,
				SpecKey:     "files",
			},
			{
				Name:        "token",
				Label:       "Hugging Face Token",
				Type:        "password",
				Placeholder: "hf_...",
				Description: "Access token for private or gated models (optional).",
				Required:    false,
				SpecKey:     "token",
			},
			{
				Name:        "target",
				Label:       "Target Mount Path",
				Type:        "text",
				Placeholder: "e.g. /workspace/models/checkpoints/",
				Description: "Absolute path inside the container where the resources are placed.",
				Required:    true,
				SpecKey:     "target",
			},
		},
		Validator: func(spec map[string]any) error {
			if url, ok := spec["url"].(string); ok && url != "" {
				matched, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`, url)
				if !matched {
					return fmt.Errorf("HF resource url must use 'org/repo' format (e.g., 'runwayml/stable-diffusion-v1-5')")
				}
			}
			return nil
		},
	},
	{
		ID:          "file",
		Name:        "Inline File / Content",
		Description: "Directly create a configuration or script file from text content.",
		Fields: []ProviderField{
			{
				Name:        "content",
				Label:       "Content",
				Type:        "textarea",
				Placeholder: "# Paste file content or configuration here...",
				Description: "Raw text content to write into the file.",
				Required:    true,
				SpecKey:     "content",
			},
			{
				Name:        "target",
				Label:       "Target Mount Path",
				Type:        "text",
				Placeholder: "e.g. /workspace/config.yaml or /app/.env",
				Description: "Absolute destination path for the created file.",
				Required:    true,
				SpecKey:     "target",
			},
			{
				Name:        "permissions",
				Label:       "Permissions",
				Type:        "text",
				Placeholder: "0644",
				Description: "File mode permissions (default 0644, or 0755 for executables).",
				Required:    false,
				SpecKey:     "permissions",
			},
		},
	},
	{
		ID:          "git",
		Name:        "Git Repository",
		Description: "Clone a repository from GitHub, GitLab, or any Git host.",
		Fields: []ProviderField{
			{
				Name:        "url",
				Label:       "Repository URL",
				Type:        "text",
				Placeholder: "https://github.com/org/repo.git",
				Description: "Direct Git clone URL.",
				Required:    true,
				SpecKey:     "url",
			},
			{
				Name:        "branch",
				Label:       "Branch / Revision",
				Type:        "text",
				Placeholder: "main",
				Description: "Branch, tag, or commit to clone.",
				Required:    false,
				SpecKey:     "revision",
			},
			{
				Name:        "token",
				Label:       "Personal Access Token",
				Type:        "password",
				Placeholder: "ghp_...",
				Description: "Access token for private repositories.",
				Required:    false,
				SpecKey:     "token",
			},
			{
				Name:        "target",
				Label:       "Target Mount Path",
				Type:        "text",
				Placeholder: "e.g. /workspace/custom-handlers",
				Description: "Absolute destination path for the cloned repository.",
				Required:    true,
				SpecKey:     "target",
			},
		},
		Validator: func(spec map[string]any) error {
			if url, ok := spec["url"].(string); ok && url != "" {
				if !strings.HasSuffix(url, ".git") && !strings.Contains(url, "github.com") && !strings.Contains(url, "gitlab.com") {
					return fmt.Errorf("GIT resource url does not look like a valid git URL")
				}
			}
			return nil
		},
	},
	{
		ID:          "s3",
		Name:        "Amazon S3 / Object Storage",
		Description: "Download files or entire folders from S3, MinIO, R2, or Wasabi.",
		Fields: []ProviderField{
			{
				Name:        "bucket",
				Label:       "Bucket Name",
				Type:        "text",
				Placeholder: "my-model-bucket",
				Description: "Name of the S3 bucket.",
				Required:    true,
				SpecKey:     "bucket",
			},
			{
				Name:        "key",
				Label:       "Object Key / Prefix",
				Type:        "text",
				Placeholder: "models/checkpoint.bin or models/",
				Description: "Key for a file or prefix ending in / for a directory.",
				Required:    true,
				SpecKey:     "key",
			},
			{
				Name:        "endpoint",
				Label:       "Endpoint URL",
				Type:        "text",
				Placeholder: "https://s3.amazonaws.com",
				Description: "Custom endpoint for MinIO, R2, or Wasabi (optional).",
				Required:    false,
				SpecKey:     "endpoint",
			},
			{
				Name:        "region",
				Label:       "Region",
				Type:        "text",
				Placeholder: "us-east-1",
				Description: "AWS region.",
				Required:    false,
				SpecKey:     "region",
			},
			{
				Name:        "access_key_id",
				Label:       "Access Key ID",
				Type:        "text",
				Placeholder: "AKIA...",
				Description: "Access key for private buckets.",
				Required:    false,
				SpecKey:     "access_key_id",
			},
			{
				Name:        "secret_access_key",
				Label:       "Secret Access Key",
				Type:        "password",
				Placeholder: "Secret access key",
				Description: "Secret key for private buckets.",
				Required:    false,
				SpecKey:     "secret_access_key",
			},
			{
				Name:        "target",
				Label:       "Target Mount Path",
				Type:        "text",
				Placeholder: "e.g. /workspace/models/s3-checkpoint",
				Description: "Absolute destination path for downloaded files.",
				Required:    true,
				SpecKey:     "target",
			},
		},
	},
}