package grpc

import (
	"context"
	"time"

	pb "github.com/russ315/kazakhexpress-protos/kazakhexpress/user/v1"
	"kazakhexpress/user-service/internal/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserGRPCHandler struct {
	pb.UnimplementedUserServiceServer
	svc user.Service
}

func NewUserGRPCHandler(svc user.Service) *UserGRPCHandler {
	return &UserGRPCHandler{svc: svc}
}

func domainToProtoUser(u *user.User) *pb.User {
	return &pb.User{
		Id:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Phone:     u.Phone,
		Address:   u.Address,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}
}

func (h *UserGRPCHandler) RegisterUser(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	input := &user.RegisterInput{
		Email:     req.GetEmail(),
		Password:  req.GetPassword(),
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Phone:     req.GetPhone(),
		Address:   req.GetAddress(),
	}

	result, err := h.svc.Register(input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}

	return &pb.AuthResponse{
		User:         domainToProtoUser(&result.User),
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserGRPCHandler) LoginUser(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	input := &user.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	result, err := h.svc.Login(input)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login failed: %v", err)
	}

	return &pb.AuthResponse{
		User:         domainToProtoUser(&result.User),
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserGRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	u, err := h.svc.GetProfile(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &pb.UserResponse{User: domainToProtoUser(u)}, nil
}

func (h *UserGRPCHandler) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.UserResponse, error) {
	return h.GetUser(ctx, &pb.GetUserRequest{UserId: req.GetUserId()})
}

func (h *UserGRPCHandler) UpdateUserProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserResponse, error) {
	input := &user.UpdateProfileInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Address:   req.Address,
	}

	u, err := h.svc.UpdateProfile(req.GetUserId(), input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}

	return &pb.UserResponse{User: domainToProtoUser(u)}, nil
}

func (h *UserGRPCHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	userID, err := h.svc.ValidateToken(req.GetToken())
	if err != nil {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: userID,
	}, nil
}

func (h *UserGRPCHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	result, err := h.svc.RefreshToken(req.GetRefreshToken())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "refresh failed: %v", err)
	}

	return &pb.AuthResponse{
		User:         domainToProtoUser(&result.User),
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *UserGRPCHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Empty, error) {
	if err := h.svc.Logout(req.GetUserId(), req.GetAccessToken(), req.GetRefreshToken()); err != nil {
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	return &pb.Empty{}, nil
}

func (h *UserGRPCHandler) ForgotPassword(ctx context.Context, req *pb.ForgotPasswordRequest) (*pb.Empty, error) {
	input := &user.ForgotPasswordInput{Email: req.GetEmail()}
	if err := h.svc.ForgotPassword(input); err != nil {
		return nil, status.Errorf(codes.Internal, "forgot password failed: %v", err)
	}

	return &pb.Empty{}, nil
}

func (h *UserGRPCHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.Empty, error) {
	input := &user.ResetPasswordInput{
		Token:       req.GetToken(),
		NewPassword: req.GetNewPassword(),
	}

	if err := h.svc.ResetPassword(input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "reset password failed: %v", err)
	}

	return &pb.Empty{}, nil
}

func (h *UserGRPCHandler) HealthCheck(ctx context.Context, _ *pb.Empty) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}

func UnixToTime(ts int64) time.Time {
	return time.Unix(ts, 0)
}
