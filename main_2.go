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
)

type Program struct {
	*aggregate.Base

	Name string
}

type CreateProgramCommand struct {
	Name string
}

func NewProgram(id uuid.UUID) *Program {
	p := &Program{Base: aggregate.New("program", id)}

	event.ApplyWith(p, p.create, "program.created")

	return p
}

func (p *Program) MarshalText() (text []byte, err error) {
	return []byte(p.Name), nil
}

func (p *Program) UnmarshalText(text []byte) error {
	p.Name = string(text)
	return nil
}

func (p *Program) Create(cmd CreateProgramCommand) error {
	// aggregate.Next() creates the event and applies it using l.ApplyEvent()
	aggregate.Next(p, "program.created", cmd)

	return nil
}

func (p *Program) create(evt event.Of[CreateProgramCommand]) {
	p.Name = evt.Data().Name
}

func main() {
	id := uuid.New()
	program := NewProgram(id)
	err := program.Create(CreateProgramCommand{"My super duper program"})
	if err != nil {
		return
	}

	c := codec.New()
	codec.Register[CreateProgramCommand](c, "program.created")

	s := mongo.NewEventStore(c, mongo.URL("mongodb://localhost:27017"), mongo.Collection("tenant_1"))
	r := repository.New(s)

	if err := r.Save(context.TODO(), program); err != nil {
		panic(fmt.Errorf("save todo list: %w", err))
	}

	fmt.Println("===================================================")

	program2 := NewProgram(id)

	err = r.Fetch(context.TODO(), program2)
	if err != nil {
		panic(fmt.Errorf(
			"fetch program list: %w [id=%s]", err, program2.AggregateID(),
		))
	}

	fmt.Println(program2)
}
