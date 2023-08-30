package actors

type Action func()
type ActionWithActor func(actor *Actor)
type ActionWithCallback func(actor *Actor) Action

type Actor struct {
	actionChan chan ActionWithActor
	Data       Data
}

func (actor *Actor) Do(action ActionWithActor) {
	actor.actionChan <- action
}

func (actor *Actor) Call(action ActionWithCallback, callbackChan chan Action) {
	actor.actionChan <- func(actor *Actor) {
		callbackChan <- action(actor)
	}
}

func (actor *Actor) DoAndCallback(action ActionWithCallback) {
	resultChanel := make(chan Action, 0)
	actor.Call(action, resultChanel)
	callback := <-resultChanel
	callback()
}

func NewActor(Data Data, chanSize uint) *Actor {
	actor := &Actor{make(chan ActionWithActor, chanSize), Data}
	go actor.actionLoop()
	return actor
}

func (actor *Actor) actionLoop() {
	for {
		action := <-actor.actionChan
		action(actor)
	}
}

type Data interface{}
