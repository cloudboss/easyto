package aws

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/cloudboss/easyto/pkg/preinit/maps"
)

type SSMClient interface {
	GetParameters(ssmPath string) (maps.ParameterMap, error)
}

type ssmClient struct {
	client *ssm.Client
}

func NewSSMClient(cfg aws.Config) SSMClient {
	return &ssmClient{
		client: ssm.NewFromConfig(cfg),
	}
}

func (s *ssmClient) GetParameters(ssmPath string) (maps.ParameterMap, error) {
	parameters, err := s.getParameters(ssmPath)
	if err != nil {
		return nil, err
	}
	return parametersToMap(parameters, ssmPath), nil
}

func (s *ssmClient) getParameters(ssmPath string) ([]types.Parameter, error) {
	var (
		parameters []types.Parameter
		nextToken  *string
	)

	for {
		out, err := s.client.GetParametersByPath(context.Background(),
			&ssm.GetParametersByPathInput{
				Path:           p(ssmPath),
				Recursive:      p(true),
				WithDecryption: p(true),
				NextToken:      nextToken,
			})
		if err != nil {
			return nil, fmt.Errorf("unable to get SSM parameters at path %s: %w", ssmPath, err)
		}
		parameters = append(parameters, out.Parameters...)
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return parameters, nil
}

func parametersToMap(parameters []types.Parameter, prefix string) maps.ParameterMap {
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	pMap := map[string]any{}
	for _, param := range parameters {
		fields := strings.Split(*param.Name, prefix)
		if len(fields) != 2 {
			continue
		}
		if strings.Contains(fields[1], "/") {
			newFields := strings.Split(fields[1], "/")
			newPrefix := filepath.Join(prefix, newFields[0])
			pMap[newFields[0]] = parametersToMap(parameters, newPrefix)
		} else {
			pMap[fields[1]] = *param.Value
		}
	}

	return pMap
}
