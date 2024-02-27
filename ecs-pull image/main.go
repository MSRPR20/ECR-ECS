package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

const (
	awsRegion         = "ap-southeast-1"                                                                  // replace with your AWS region
	clusterName       = "cluster name"                                                                    // replace with your desired ECS cluster name
	taskDefName       = "task-def"                                                                        // replace with your desired task definition name
	containerName     = "container-name"                                                                  // replace with your desired container name
	image             = "nginx:latest"                                                                    // replace with your Docker image URL
	ecsServiceName    = "service-name"                                                                    // replace with your desired ECS service name
	desiredTaskCount  = 1                                                                                 // replace with the desired number of tasks to run
	loadBalancerName  = "arn:aws:elasticloadbalancing:region-1:account-id:loadbalancer/loadbalancer-name" // replace with your desired load balancer name
	loadBalancerPort  = 80                                                                                // replace with the desired load balancer port
	containerPort     = 80                                                                                // replace with the desired container port
	securityGroupName = "security goup"                                                                   // replace with your desired security group name
	Memory            = "512"
	requires          = "FARGATE"
	networkmode       = "awsvpc"
	CPU               = "256"
	executionrole     = "arn:aws:iam::891377012705:role/execution role"
	taskrole          = "arn:aws:iam::891377012705:role/task-role"
	tcp               = "tcp"
	subnetId          = "subnet-id"
)

func main() {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		os.Exit(1)
	}

	// Create ECS cluster
	clusterArn, err := useECSCluster(ctx, cfg)
	if err != nil {
		fmt.Println("Error creating ECS cluster:", err)
		os.Exit(1)
	}
	fmt.Println("ECS cluster created successfully:", clusterArn)

	// Register ECS task definition
	taskDefArn, err := registerTaskDefinition(ctx, cfg)
	if err != nil {
		fmt.Println("Error registering ECS task definition:", err)
		os.Exit(1)
	}
	fmt.Println("ECS task definition registered successfully:", taskDefArn)

	// Run ECS service with EC2 launch type
	serviceArn, err := runECSService(ctx, cfg, clusterName, taskDefArn, ecsServiceName, desiredTaskCount)
	if err != nil {
		fmt.Println("Error running ECS service:", err)
		os.Exit(1)
	}
	fmt.Println("ECS service started successfully:", serviceArn)

	// Create Application Load Balancer
	loadBalancerArn, err := useExistingLoadBalancer(ctx, cfg, loadBalancerName, loadBalancerPort)
	if err != nil {
		fmt.Println("Error creating Application Load Balancer:", err)
		os.Exit(1)
	}
	fmt.Println("Application Load Balancer created successfully:", loadBalancerArn)
}

func useECSCluster(ctx context.Context, cfg aws.Config) (string, error) {
	ecsClient := ecs.NewFromConfig(cfg)

	// Describe ECS clusters to check if the cluster already exists
	describeClustersOutput, err := ecsClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterName},
	})
	if err != nil {
		return "", err
	}

	return *describeClustersOutput.Clusters[0].ClusterArn, nil

}

func registerTaskDefinition(ctx context.Context, cfg aws.Config) (string, error) {
	ecsClient := ecs.NewFromConfig(cfg)

	// Register ECS task definition
	containerDefinition := types.ContainerDefinition{
		Name:  aws.String(containerName),
		Image: aws.String(image),
		PortMappings: []types.PortMapping{
			{
				ContainerPort: aws.Int32(containerPort),
				HostPort:      aws.Int32(containerPort),
			},
		},
	}

	registerTaskDefinitionInput := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    []types.ContainerDefinition{containerDefinition},
		Family:                  aws.String(taskDefName),
		RequiresCompatibilities: []types.Compatibility{requires},
		NetworkMode:             types.NetworkModeAwsvpc,
		Cpu:                     aws.String(CPU),
		Memory:                  aws.String(Memory),
		ExecutionRoleArn:        aws.String(executionrole),
		TaskRoleArn:             aws.String(taskrole),
	}

	registerTaskDefinitionOutput, err := ecsClient.RegisterTaskDefinition(ctx, registerTaskDefinitionInput)
	if err != nil {
		return "", err
	}

	return *registerTaskDefinitionOutput.TaskDefinition.TaskDefinitionArn, nil
}

func runECSService(ctx context.Context, cfg aws.Config, clusterName, taskDefArn, serviceName string, desiredCount int32) (string, error) {
	ecsClient := ecs.NewFromConfig(cfg)

	// Create ECS service
	createServiceOutput, err := ecsClient.CreateService(ctx, &ecs.CreateServiceInput{
		Cluster:        aws.String(clusterName),
		ServiceName:    aws.String(serviceName),
		TaskDefinition: aws.String(taskDefArn),
		DesiredCount:   aws.Int32(desiredCount),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{

				Subnets:        []string{subnetId},
				SecurityGroups: []string{securityGroupName},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return *createServiceOutput.Service.ServiceArn, nil
}

func useExistingLoadBalancer(ctx context.Context, cfg aws.Config, lbArnOrName string, lbPort int32) (string, error) {
	elbclient := elasticloadbalancingv2.NewFromConfig(cfg)

	// Describe the existing load balancer to check if it already exists
	describeLoadBalancersOutput, err := elbclient.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{lbArnOrName},
	})
	if err != nil {
		return "", err
	}

	// Check if the load balancer already exists
	if len(describeLoadBalancersOutput.LoadBalancers) > 0 {
		return *describeLoadBalancersOutput.LoadBalancers[0].LoadBalancerArn, nil
	}

	// If the load balancer doesn't exist, return an error
	return "", fmt.Errorf("load balancer with ARN or name %s not found", lbArnOrName)
}
