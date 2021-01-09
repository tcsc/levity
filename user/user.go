package user

import "context"

// User encapsulates an authenticated user in the system. For the purposes
// of this example it is simply the user's login name, but it's pretty easy
// to imagine it being expanded to include roles, permissions, etc.
//
// Once created, a `User` is immutable (hence no locking)
type User struct {
	login string
}

// Login fetches the user's login name
func (user *User) Login() string {
	return user.login
}

// Is tests if two user objects refer to the same underlying user.
func (user *User) Is(other *User) bool {
	// if they are literally the same object, then yep, they do
	if user == other {
		return true
	}

	return user.login == other.login
}

// New initialises a user record with
func New(login string) *User {
	return &User{login: login}
}

type userKey struct{}

// NewContext creates a context with a user record attached.
func NewContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

// FromContext retrieves a user record from the supplied context.
func FromContext(ctx context.Context) (u *User, ok bool) {
	u, ok = ctx.Value(userKey{}).(*User)
	return
}

// MustFromContext retrieves a user record from the supplied context.
func MustFromContext(ctx context.Context) *User {
	u, ok := FromContext(ctx)
	if !ok {
		panic("No user record found in context")
	}
	return u
}
