package matching

import "errors"

var (
	ErrEmptyUserID = errors.New("user ID must not be empty")
	ErrSameUser    = errors.New("pair requires two different users")
)

// type UserID string にしないのは、private な value を持つ UserID として実装すると
// matching.UserID("") のように直接 validation を通さないで UserID を作れてしまうため、NewUserID を必ず経由しないと作れないような作りにするため
type UserID struct {
	value string
}

func NewUserID(value string) (UserID, error) {
	if value == "" {
		return UserID{}, ErrEmptyUserID
	}

	return UserID{value: value}, nil
}

func (id UserID) String() string {
	return id.value
}

type Pair struct {
	low  UserID
	high UserID
}

func NewPair(first, second UserID) (Pair, error) {
	if first.value == "" || second.value == "" {
		return Pair{}, ErrEmptyUserID
	}

	if first == second {
		return Pair{}, ErrSameUser
	}

	if first.value > second.value {
		first, second = second, first
	}

	return Pair{
		low:  first,
		high: second,
	}, nil
}

func (p Pair) Low() UserID {
	return p.low
}

func (p Pair) High() UserID {
	return p.high
}
