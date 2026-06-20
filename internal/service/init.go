package service

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewUserService),
	fx.Provide(NewEmailService),
	fx.Provide(NewHealthService),
	fx.Provide(NewLocalAuthService),
	fx.Provide(NewTaskService),
	fx.Provide(NewChatService),
	fx.Provide(NewWorkspaceService),
	fx.Provide(NewWorkspaceInvitationService),
	fx.Provide(NewWorkspaceMemberService),
)
