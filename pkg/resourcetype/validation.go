package resourcetype

import (
	"fmt"
	"strings"
)

func ValidateResource(resourceType string, spec map[string]any) error {
	provider := findProvider(resourceType)
	if provider == nil {
		return fmt.Errorf("unsupported resource type %q", resourceType)
	}

	for _, field := range provider.Fields {
		if !field.Required {
			continue
		}

		specKey := field.SpecKey
		if specKey == "" {
			specKey = strings.ToLower(field.Name)
		}

		value := spec[specKey]
		if value == nil {
			return fmt.Errorf("resource %q is missing required field %q", resourceType, field.Label)
		}

		strVal, ok := value.(string)
		if ok && strings.TrimSpace(strVal) == "" {
			return fmt.Errorf("resource %q field %q cannot be empty", resourceType, field.Label)
		}

		if field.Name == "content" && field.Required {
			if !ok || strings.TrimSpace(strVal) == "" {
				return fmt.Errorf("resource %q requires non-empty content", resourceType)
			}
		}
	}

	if provider.Validator != nil {
		if err := provider.Validator(spec); err != nil {
			return err
		}
	}

	return nil
}

func findProvider(id string) *Provider {
	for _, p := range Providers {
		if strings.EqualFold(p.ID, id) {
			return &p
		}
	}
	return nil
}

func FindProvider(id string) *Provider {
	return findProvider(id)
}

func GetValidResourceTypes() []string {
	types := make([]string, len(Providers))
	for i, p := range Providers {
		types[i] = p.ID
	}
	return types
}