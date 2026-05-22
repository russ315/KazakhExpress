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
	GetUser(ctx context.Context, userID string) (User, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	UpdateProfile(ctx context.Context, userID string, input UpdateProfileRequest) (User, error)
	ValidateToken(ctx context.Context, token string) (ValidateTokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error)
	Logout(ctx context.Context, input LogoutRequest) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx, &userv1.Empty{})
	return err
}

func (c *GRPCClient) Register(ctx context.Context, input RegisterRequest) (AuthResponse, error) {
	resp, err := c.client.RegisterUser(ctx, &userv1.RegisterRequest{
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Phone:     input.Phone,
		Address:   input.Address,
	})
	if err != nil {
		return AuthResponse{}, err
	}
	return authResponseFromProto(resp), nil
}

func (c *GRPCClient) Login(ctx context.Context, input LoginRequest) (AuthResponse, error) {
	resp, err := c.client.LoginUser(ctx, &userv1.LoginRequest{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return AuthResponse{}, err
	}
	return authResponseFromProto(resp), nil
}

func (c *GRPCClient) GetUser(ctx context.Context, userID string) (User, error) {
	resp, err := c.client.GetUser(ctx, &userv1.GetUserRequest{UserId: userID})
	if err != nil {
		return User{}, err
	}
	return userFromProto(resp.GetUser()), nil
}

func (c *GRPCClient) GetUserByID(ctx context.Context, userID string) (User, error) {
	resp, err := c.client.GetUserByID(ctx, &userv1.GetUserByIDRequest{UserId: userID})
	if err != nil {
		return User{}, err
	}
	return userFromProto(resp.GetUser()), nil
}

func (c *GRPCClient) UpdateProfile(ctx context.Context, userID string, input UpdateProfileRequest) (User, error) {
	resp, err := c.client.UpdateUserProfile(ctx, &userv1.UpdateProfileRequest{
		UserId:    userID,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Phone:     input.Phone,
		Address:   input.Address,
	})
	if err != nil {
		return User{}, err
	}
	return userFromProto(resp.GetUser()), nil
}

func (c *GRPCClient) ValidateToken(ctx context.Context, token string) (ValidateTokenResponse, error) {
	resp, err := c.client.ValidateToken(ctx, &userv1.ValidateTokenRequest{Token: token})
	if err != nil {
		return ValidateTokenResponse{}, err
	}
	return ValidateTokenResponse{
		Valid:  resp.GetValid(),
		UserID: resp.GetUserId(),
	}, nil
}

func (c *GRPCClient) RefreshToken(ctx context.Context, refreshToken string) (AuthResponse, error) {
	resp, err := c.client.RefreshToken(ctx, &userv1.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return AuthResponse{}, err
	}
	return authResponseFromProto(resp), nil
}

func (c *GRPCClient) Logout(ctx context.Context, input LogoutRequest) error {
	_, err := c.client.Logout(ctx, &userv1.LogoutRequest{
		UserId:       input.UserID,
		AccessToken:  input.AccessToken,
		RefreshToken: input.RefreshToken,
	})
	return err
}

func (c *GRPCClient) ForgotPassword(ctx context.Context, email string) error {
	_, err := c.client.ForgotPassword(ctx, &userv1.ForgotPasswordRequest{Email: email})
	return err
}

func (c *GRPCClient) ResetPassword(ctx context.Context, token, newPassword string) error {
	_, err := c.client.ResetPassword(ctx, &userv1.ResetPasswordRequest{
		Token:       token,
		NewPassword: newPassword,
	})
	return err
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
	return &GRPCClient{
		conn:   conn,
		client: userv1.NewUserServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func userFromProto(item *userv1.User) User {
	if item == nil {
		return User{}
	}
	return User{
		ID:        item.GetId(),
		Email:     item.GetEmail(),
		FirstName: item.GetFirstName(),
		LastName:  item.GetLastName(),
		Phone:     item.GetPhone(),
		Address:   item.GetAddress(),
		CreatedAt: item.GetCreatedAt(),
		UpdatedAt: item.GetUpdatedAt(),
	}
}

func authResponseFromProto(item *userv1.AuthResponse) AuthResponse {
	if item == nil {
		return AuthResponse{}
	}
	return AuthResponse{
		User:         userFromProto(item.GetUser()),
		Token:        item.GetToken(),
		RefreshToken: item.GetRefreshToken(),
	}
}
