package domain

import "github.com/bigphu/goat/errs"

// Lỗi của identity. type là định danh ổn định cho client; detail là câu tiếng Việt
// hiển thị thẳng cho người dùng.
var (
	// Sai email hay sai mật khẩu đều là lỗi này, để không dò được email nào tồn tại
	ErrBadCredentials = errs.Unauthorized("Email hoặc mật khẩu không đúng.",
		errs.WithType("/errors/bad-credentials"), errs.WithTitle("Bad credentials"))

	ErrInvalidEmail = errs.Unprocessable("Email không hợp lệ.",
		errs.WithType("/errors/invalid-email"), errs.WithTitle("Invalid email"),
		errs.WithFields(errs.FieldInvalid("email", "Email không hợp lệ.")))

	ErrInvalidName = errs.Unprocessable("Họ và tên cần từ 1 đến 100 ký tự.",
		errs.WithType("/errors/invalid-name"), errs.WithTitle("Invalid name"),
		errs.WithFields(errs.FieldInvalid("name", "Họ và tên cần từ 1 đến 100 ký tự.")))

	ErrInvalidUnit = errs.Unprocessable("Số căn hộ tối đa 20 ký tự.",
		errs.WithType("/errors/invalid-unit"), errs.WithTitle("Invalid unit"),
		errs.WithFields(errs.FieldInvalid("unit", "Số căn hộ tối đa 20 ký tự.")))

	ErrInvalidRole = errs.Unprocessable("Vai trò không hợp lệ.",
		errs.WithType("/errors/invalid-role"), errs.WithTitle("Invalid role"))

	ErrWeakPassword = errs.Unprocessable("Mật khẩu cần ít nhất 8 ký tự.",
		errs.WithType("/errors/weak-password"), errs.WithTitle("Password does not meet the policy"),
		errs.WithFields(errs.FieldInvalid("password", "Mật khẩu cần ít nhất 8 ký tự.")))

	ErrPasswordIsEmail = errs.Unprocessable("Mật khẩu không được trùng với email.",
		errs.WithType("/errors/password-is-email"), errs.WithTitle("Password equals email"),
		errs.WithFields(errs.FieldInvalid("password", "Mật khẩu không được trùng với email.")))

	// Không dùng 401: người dùng đang đăng nhập hợp lệ
	ErrWrongPassword = errs.Unprocessable("Mật khẩu hiện tại không đúng.",
		errs.WithType("/errors/wrong-password"), errs.WithTitle("Current password is incorrect"),
		errs.WithFields(errs.FieldInvalid("current_password", "Mật khẩu hiện tại không đúng.")))

	ErrEmailTaken = errs.Conflict("Email này đã có tài khoản.",
		errs.WithType("/errors/email-taken"), errs.WithTitle("Email already in use"))

	ErrAccountNotFound = errs.NotFound("Không tìm thấy tài khoản.",
		errs.WithType("/errors/account-not-found"), errs.WithTitle("Account not found"))

	ErrNotInvited = errs.Conflict("Chỉ gửi lại lời mời cho tài khoản chưa kích hoạt.",
		errs.WithType("/errors/not-invited"), errs.WithTitle("Account is not waiting for an invitation"))

	ErrSelfChange = errs.Conflict("Bạn không thể khoá hoặc đổi vai trò của chính mình.",
		errs.WithType("/errors/self-change"), errs.WithTitle("Cannot change own account"))

	// Link mời / đặt lại đã dùng, bị thay hoặc hết hạn
	ErrInvalidPasswordToken = errs.Unprocessable("Liên kết đã hết hạn hoặc không còn hiệu lực. Hãy yêu cầu liên kết mới.",
		errs.WithType("/errors/invalid-password-token"), errs.WithTitle("Invalid or expired link"))
)
