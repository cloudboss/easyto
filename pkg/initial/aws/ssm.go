package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/cloudboss/easyto/pkg/initial/collections"
)

type ssmAPI interface {
	GetParametersByPath(context.Context, *ssm.GetParametersByPathInput,
		...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	GetParameter(context.Context, *ssm.GetParameterInput,
		...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type SSMClient interface {
	GetParameterList(ssmPath string) (collections.WritableList, error)
	GetParameterMap(ssmPath string) (map[string]string, error)
	GetParameterValue(ssmPath string) ([]byte, error)
}

type ssmClient struct {
	api ssmAPI
}

func NewSSMClient(cfg aws.Config) SSMClient {
	return &ssmClient{
		api: ssm.NewFromConfig(cfg),
	}
}

func (s *ssmClient) GetParameterList(ssmPath string) (collections.WritableList, error) {
	parameters, err := s.getParameters(ssmPath)
	if err != nil {
		return nil, err
	}
	return s.toList(parameters, ssmPath)
}

func (s *ssmClient) GetParameterMap(ssmPath string) (map[string]string, error) {
	parameter, err := s.getParameter(ssmPath)
	if err != nil {
		return nil, err
	}
	var r io.Reader
	r = strings.NewReader(*parameter.Value)
	m := make(map[string]string)
	err = json.NewDecoder(r).Decode(&m)
	if err != nil {
		err = fmt.Errorf("unable to decode map from parameter %s: %w", ssmPath, err)
		return nil, err
	}
	return m, nil
}

func (s *ssmClient) GetParameterValue(ssmPath string) ([]byte, error) {
	parameter, err := s.getParameter(ssmPath)
	if err != nil {
		return nil, err
	}
	return []byte(*parameter.Value), nil
}

func (s *ssmClient) getParameters(ssmPath string) ([]types.Parameter, error) {
	var (
		parameters []types.Parameter
		err        error
	)
	if strings.HasPrefix(ssmPath, "/") {
		parameters, err = s.getParametersByPath(ssmPath)
		if err != nil {
			return nil, err
		}
	}
	if len(parameters) == 0 {
		parameter, err := s.getParameter(ssmPath)
		if err != nil {
			return nil, err
		}
		parameters = []types.Parameter{*parameter}
	}
	return parameters, nil
}

func (s *ssmClient) getParametersByPath(ssmPath string) ([]types.Parameter, error) {
	var (
		parameters []types.Parameter
		nextToken  *string
	)
	for {
		out, err := s.api.GetParametersByPath(context.Background(),
			&ssm.GetParametersByPathInput{
				Path:           p(ssmPath),
				Recursive:      p(true),
				WithDecryption: p(true),
				NextToken:      nextToken,
			})
		if err != nil {
			return nil, fmt.Errorf("unable to get SSM parameters at path %s: %w",
				ssmPath, err)
		}
		parameters = append(parameters, out.Parameters...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return parameters, nil
}

func (s *ssmClient) getParameter(ssmPath string) (*types.Parameter, error) {
	out, err := s.api.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           p(ssmPath),
		WithDecryption: p(true),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get SSM parameter %s: %w", ssmPath, err)
	}
	return out.Parameter, nil
}

func (s *ssmClient) toList(parameters []types.Parameter, ssmPath string) (collections.WritableList, error) {
	list := collections.WritableList{}
	for _, parameter := range parameters {
		if !strings.HasPrefix(*parameter.Name, ssmPath) {
			continue
		}
		name := *parameter.Name
		if len(ssmPath) > 0 {
			fields := strings.Split(name, ssmPath)
			name = fields[1]
		}
		valueRC := io.NopCloser(strings.NewReader(*parameter.Value))
		listEntry := &collections.WritableListEntry{Path: name, Value: valueRC}
		list = append(list, listEntry)
	}
	return list, nil
}
