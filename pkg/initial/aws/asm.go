package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	asm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/cloudboss/easyto/pkg/initial/collections"
)

type asmAPI interface {
	GetSecretValue(context.Context, *asm.GetSecretValueInput,
		...func(*asm.Options)) (*asm.GetSecretValueOutput, error)
}

type ASMClient interface {
	// GetSecretList retrieves only one item but returns a
	// WritableList, for consistency with the other AWS clients,
	// and since it has the desired behavior for writing to disk.
	GetSecretList(secretID string) (collections.WritableList, error)
	GetSecretMap(secretID string) (map[string]string, error)
	GetSecretValue(secretID string) ([]byte, error)
}

type asmClient struct {
	api asmAPI
}

func NewASMClient(cfg aws.Config) ASMClient {
	return &asmClient{
		api: asm.NewFromConfig(cfg),
	}
}

func (a *asmClient) GetSecretList(secretID string) (collections.WritableList, error) {
	secret, err := a.getSecret(secretID)
	if err != nil {
		return nil, err
	}
	return a.toList(secret)
}

func (a *asmClient) GetSecretMap(secretID string) (map[string]string, error) {
	secret, err := a.getSecret(secretID)
	if err != nil {
		return nil, err
	}
	var r io.Reader
	if secret.SecretString != nil {
		r = strings.NewReader(*secret.SecretString)
	} else if secret.SecretBinary != nil {
		r = bytes.NewReader(secret.SecretBinary)
	}
	m := make(map[string]string)
	err = json.NewDecoder(r).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("unable to decode map from secret %s: %w", secretID, err)
	}
	return m, nil
}

func (a *asmClient) GetSecretValue(secretID string) ([]byte, error) {
	secret, err := a.getSecret(secretID)
	if err != nil {
		return nil, err
	}
	var value []byte
	if secret.SecretString != nil {
		value = []byte(*secret.SecretString)
	} else if secret.SecretBinary != nil {
		value = secret.SecretBinary
	}
	return value, nil
}

func (a *asmClient) getSecret(secretID string) (*asm.GetSecretValueOutput, error) {
	secret, err := a.api.GetSecretValue(context.Background(), &asm.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %s: %w", secretID, err)
	}
	if secret == nil {
		return nil, nil
	}
	if secret.SecretString == nil && secret.SecretBinary == nil {
		return nil, fmt.Errorf("secret %s has no value", secretID)
	}
	return secret, nil
}

func (a *asmClient) toList(secret *asm.GetSecretValueOutput) (collections.WritableList, error) {
	value := &collections.WritableListEntry{}
	if secret.SecretString != nil {
		valueRC := io.NopCloser(strings.NewReader(*secret.SecretString))
		value.Value = valueRC
	} else if secret.SecretBinary != nil {
		valueRC := io.NopCloser(bytes.NewReader(secret.SecretBinary))
		value.Value = valueRC
	}
	return collections.WritableList{value}, nil
}
