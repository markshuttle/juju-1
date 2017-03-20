// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodel_test

import (
	"github.com/juju/errors"
	jtesting "github.com/juju/testing"
	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/names.v2"

	"github.com/juju/juju/apiserver/crossmodel"
	jujucrossmodel "github.com/juju/juju/core/crossmodel"
)

const (
	addOfferCall    = "addOfferCall"
	updateOfferCall = "updateOfferCall"
	listOffersCall  = "listOffersCall"
	removeOfferCall = "removeOfferCall"
)

type mockApplicationOffers struct {
	jtesting.Stub

	addOffer   func(offer jujucrossmodel.AddApplicationOfferArgs) (*jujucrossmodel.ApplicationOffer, error)
	listOffers func(filters ...jujucrossmodel.ApplicationOfferFilter) ([]jujucrossmodel.ApplicationOffer, error)
}

func (m *mockApplicationOffers) AddOffer(offer jujucrossmodel.AddApplicationOfferArgs) (*jujucrossmodel.ApplicationOffer, error) {
	m.AddCall(addOfferCall)
	return m.addOffer(offer)
}

func (m *mockApplicationOffers) ListOffers(filters ...jujucrossmodel.ApplicationOfferFilter) ([]jujucrossmodel.ApplicationOffer, error) {
	m.AddCall(listOffersCall)
	return m.listOffers(filters...)
}

func (m *mockApplicationOffers) UpdateOffer(offer jujucrossmodel.AddApplicationOfferArgs) (*jujucrossmodel.ApplicationOffer, error) {
	m.AddCall(updateOfferCall)
	panic("not implemented")
}

func (m *mockApplicationOffers) Remove(url string) error {
	m.AddCall(removeOfferCall)
	panic("not implemented")
}

type mockModel struct {
	uuid  string
	name  string
	owner string
}

func (m *mockModel) UUID() string {
	return m.uuid
}

func (m *mockModel) Name() string {
	return m.name
}

func (m *mockModel) Owner() names.UserTag {
	return names.NewUserTag(m.owner)
}

type mockUserModel struct {
	model crossmodel.Model
}

func (m *mockUserModel) Model() crossmodel.Model {
	return m.model
}

type mockCharm struct {
	meta *charm.Meta
}

func (m *mockCharm) Meta() *charm.Meta {
	return m.meta
}

type mockApplication struct {
	charm *mockCharm
}

func (m *mockApplication) Charm() (crossmodel.Charm, bool, error) {
	return m.charm, true, nil
}

type mockState struct {
	modelUUID    string
	model        crossmodel.Model
	usermodels   []crossmodel.UserModel
	applications map[string]crossmodel.Application
}

func (m *mockState) Application(name string) (crossmodel.Application, error) {
	app, ok := m.applications[name]
	if !ok {
		return nil, errors.NotFoundf("application %q", name)
	}
	return app, nil
}

func (m *mockState) Model() (crossmodel.Model, error) {
	return m.model, nil
}

func (m *mockState) ModelUUID() string {
	return m.modelUUID
}

func (m *mockState) ModelsForUser(user names.UserTag) ([]crossmodel.UserModel, error) {
	return m.usermodels, nil
}

type mockStatePool struct {
	st crossmodel.Backend
}

func (st *mockStatePool) Get(modelUUID string) (crossmodel.Backend, func(), error) {
	return st.st, func() {}, nil
}
