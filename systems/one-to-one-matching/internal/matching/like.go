package matching

import "errors"

var (
	ErrSelfLike = errors.New("self like is not allow")
)

type Like struct {
	sender   UserID
	receiver UserID
}

func NewLike(sender, receiver UserID) (Like, error) {
	if sender.value == "" || receiver.value == "" {
		return Like{}, ErrEmptyUserID
	}

	if sender == receiver {
		return Like{}, ErrSelfLike
	}

	return Like{
		sender:   sender,
		receiver: receiver,
	}, nil
}

func (l Like) Sender() UserID {
	return l.sender
}

func (l Like) Receiver() UserID {
	return l.receiver
}
