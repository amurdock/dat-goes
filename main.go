package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/backend/mongo"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/event"
	"os"
	"strings"
)

type List struct {
	*aggregate.Base

	Tasks []string
}

func NewList(id uuid.UUID) *List {
	l := &List{Base: aggregate.New("list", id)}

	event.ApplyWith(l, l.addTask, "task_added")
	event.ApplyWith(l, l.removeTask, "task_removed")

	return l
}

func (l *List) MarshalText() (text []byte, err error) {
	return []byte(strings.Join(l.Tasks, ",")), nil
}

func (l *List) UnmarshalText(text []byte) error {
	l.Tasks = strings.Split(string(text), ",")
	return nil
}

func (l *List) AddTask(task string) error {
	if l.Contains(task) {
		return fmt.Errorf("list already contains %q", task)
	}

	// aggregate.Next() creates the event and applies it using l.ApplyEvent()
	aggregate.Next(l, "task_added", task)

	return nil
}

func (l *List) RemoveTask(task string) error {
	if !l.Contains(task) {
		return fmt.Errorf("list does not contain %q", task)
	}

	aggregate.Next(l, "task_removed", task)

	return nil
}

// Contains returns whether the list contains the given task.
func (l *List) Contains(task string) bool {
	task = strings.ToLower(task)
	for _, t := range l.Tasks {
		if strings.ToLower(t) == task {
			return true
		}
	}
	return false
}

func (l *List) addTask(evt event.Of[string]) {
	fmt.Println("addTask", evt)
	l.Tasks = append(l.Tasks, evt.Data())
}

func (l *List) removeTask(evt event.Of[string]) {
	fmt.Println("removeTask", evt)
	name := evt.Data()
	for i, task := range l.Tasks {
		if task == name {
			l.Tasks = append(l.Tasks[:i], l.Tasks[i+1:]...)
			return
		}
	}
}

type ListSnapshot struct{}

func (ListSnapshot) Test(aggregate.Aggregate) bool {
	return true
}

func main() {
	id := uuid.New()
	fmt.Println(id.String())
	list := NewList(id)
	err := list.AddTask("foo")
	if err != nil {
		return
	}
	err = list.AddTask("bar")
	if err != nil {
		return
	}
	err = list.AddTask("baz")
	if err != nil {
		return
	}
	err = list.RemoveTask("bar")
	if err != nil {
	}

	// err = os.Setenv("POSTGRES_EVENTSTORE", "postgres://postgres:postgres@localhost:5432/postgres")
	err = os.Setenv("MONGO_URL", "mongodb://localhost:27017")
	if err != nil {
		return
	}

	c := codec.New()
	codec.Register[string](c, "task_added")
	codec.Register[string](c, "task_removed")

	//s := mongo.NewEventStore(c, mongo.Transactions(true))
	s := mongo.NewEventStore(c)
	r := repository.New(s, repository.WithSnapshots(mongo.NewSnapshotStore(), ListSnapshot{}))
	//r := repository.New(s)

	if err := r.Save(context.TODO(), list); err != nil {
		panic(fmt.Errorf("save todo list: %w", err))
	}

	//id, _ := uuid.Parse("7022bd4f-ab35-4775-9f09-bfe2ca98e4b4")
	l := NewList(id)
	if err := r.Use(context.TODO(), l, func() error {
		_ = l.AddTask("yada")
		_ = l.AddTask("mada")
		_ = l.AddTask("bada")
		_ = l.AddTask("tada")
		return l.AddTask("lada")
	}); err != nil {
		panic(fmt.Errorf(
			"fetch todo list: %w [id=%s]", err, l.AggregateID(),
		))
	}

	fmt.Println(l)

	fmt.Println("===================================================")

	//id, _ := uuid.Parse("3c67b64e-b2a0-4383-9edb-aa54392cc903")
	list = NewList(id)

	err = r.Fetch(context.TODO(), list)
	if err != nil {
		panic(fmt.Errorf(
			"fetch todo list: %w [id=%s]", err, list.AggregateID(),
		))
	}

	fmt.Println(list)
}
