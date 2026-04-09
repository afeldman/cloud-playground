package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func fetchSSMParams(path string) ([]Parameter, error) {
	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := ssm.NewFromConfig(cfg)

	var nextToken *string
	var result []Parameter

	for {
		out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(path),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true), // 🔑 wichtig
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, p := range out.Parameters {
			key, err := NormalizeKey(path, *p.Name)
			if err != nil {
				return nil, err
			}

			val := ""
			if p.Value != nil {
				val = *p.Value
			}

			result = append(result, Parameter{
				Key:   key,
				Value: val,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return result, nil
}
