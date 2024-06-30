package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	asm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/cloudboss/easyto/pkg/initial/maps"
)

var (
	ErrSecretNameRequired       = errors.New("secret name is required")
	ErrSecretNameRequiredNotMap = errors.New("name is required when secret is not a map")
)

type ASMClient interface {
	GetSecret(secretID string, isMap bool) (maps.ParameterMap, error)
}

type asmClient struct {
	client *asm.Client
}

func NewASMClient(cfg aws.Config) ASMClient {
	return &asmClient{
		client: asm.NewFromConfig(cfg),
	}
}

func (s *asmClient) GetSecret(secretID string, isMap bool) (maps.ParameterMap, error) {
	out, err := s.client.GetSecretValue(context.Background(),
		&asm.GetSecretValueInput{
			SecretId: &secretID,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to get Secret with ID %s: %w", secretID, err)
	}
	if isMap {
		return secretMapToMap(out)
	}
	return secretToMap(out)
}

func secretToMap(secret *asm.GetSecretValueOutput) (maps.ParameterMap, error) {
	if secret == nil {
		return nil, nil
	}
	if secret.Name == nil {
		return nil, ErrSecretNameRequired
	}
	m := make(map[string]any)
	if secret.SecretString != nil {
		m["value"] = *secret.SecretString
	} else if secret.SecretBinary != nil {
		m["value"] = string(secret.SecretBinary)
	} else {
		return nil, fmt.Errorf("unable to get value of Secret with ID %s", *secret.Name)
	}
	return m, nil
}

func secretMapToMap(secret *asm.GetSecretValueOutput) (maps.ParameterMap, error) {
	if secret == nil {
		return nil, nil
	}
	if secret.Name == nil {
		return nil, ErrSecretNameRequired
	}
	var reader io.Reader
	if secret.SecretString != nil {
		reader = strings.NewReader(*secret.SecretString)
	} else if secret.SecretBinary != nil {
		reader = bytes.NewReader(secret.SecretBinary)
	} else {
		return nil, fmt.Errorf("unable to get value of Secret with ID %s", *secret.Name)
	}
	m := make(map[string]any)
	err := json.NewDecoder(reader).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("unable to decode Secret with ID %s: %w", *secret.Name, err)
	}
	return maps.ParameterMap(m), nil
}

func mapStringToMapAny(ms map[string]string) map[string]any {
	ma := make(map[string]any)
	for k, v := range ms {
		ma[k] = v
	}
	return ma
}
