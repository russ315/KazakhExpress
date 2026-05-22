package userservice

import (
	"context"
	"fmt"

	userv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Health(ctx context.Context) error
	Register(ctx context.Context, input RegisterRequest) (AuthResponse, error)
	Login(ctx context.Context, input LoginRequest) (AuthResponse, error)
	Refresh(ctx context.Context, token string) (AuthResponse, error)
	Logout(ctx context.Context, input LogoutRequest) error
	GetUser(ctx context.Context, userID string) (User, error)
	UpdateProfile(ctx context.Context, input UpdateProfileRequest) (User, error)
}

type GRPCClient struct {
	conn   *grpc.ClientConn
	client userv1.UserServiceClient
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create user grpc client: %w", err)
	}
	return &GRPCClient{conn: conn, client: userv1.NewUserServiceClient(conn)}, nil
}

func (c *GRPCClient) Close() error { return c.conn.Close() }

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx, &userv1.HealthCheckRequest{})
	return err
}

func (c *GRPCClient) Register(ctx context.Context, input RegisterRequest) (AuthResponse, error) {
	output, err := c.client.RegisterUser(ctx, &userv1.RegisterUserRequest{Email: input.Email, Password: input.Password, FirstName: input.FirstName, LastName: input.LastName, Phone: input.Phone, Address: input.Address})
	return authFromProto(output), err
}

func (c *GRPCClient) Login(ctx context.Context, input LoginRequest) (AuthResponse, error) {
	output, err := c.client.LoginUser(ctx, &userv1.LoginUserRequest{Email: input.Email, Password: input.Password})
	return authFromProto(output), err
}

func (c *GRPCClient) Refresh(ctx context.Context, token string) (AuthResponse, error) {
	output, err := c.client.RefreshToken(ctx, &userv1.RefreshTokenRequest{RefreshToken: token})
	return authFromProto(output), err
}

func (c *GRPCClient) Logout(ctx context.Context, input LogoutRequest) error {
	_, err := c.client.Logout(ctx, &userv1.LogoutRequest{UserId: input.UserID, AccessToken: input.AccessToken, RefreshToken: input.RefreshToken})
	return err
}

func (c *GRPCClient) GetUser(ctx context.Context, userID string) (User, error) {
	output, err := c.client.GetUserByID(ctx, &userv1.GetUserByIDRequest{UserId: userID})
	if err != nil {
		return User{}, err
	}
	return userFromProto(output.GetUser()), nil
}

func (c *GRPCClient) UpdateProfile(ctx context.Context, input UpdateProfileRequest) (User, error) {
	output, err := c.client.UpdateUserProfile(ctx, &userv1.UpdateUserProfileRequest{UserId: input.UserID, FirstName: input.FirstName, LastName: input.LastName, Phone: input.Phone, Address: input.Address})
	if err != nil {
		return User{}, err
	}
	return userFromProto(output.GetUser()), nil
}

func authFromProto(item *userv1.AuthResponse) AuthResponse {
	if item == nil {
		return AuthResponse{}
	}
	return AuthResponse{User: userFromProto(item.GetUser()), Token: item.GetToken(), RefreshToken: item.GetRefreshToken()}
}

func userFromProto(item *userv1.User) User {
	if item == nil {
		return User{}
	}
	return User{ID: item.GetId(), Email: item.GetEmail(), FirstName: item.GetFirstName(), LastName: item.GetLastName(), Phone: item.GetPhone(), Address: item.GetAddress(), CreatedAt: item.GetCreatedAt(), UpdatedAt: item.GetUpdatedAt()}
}
