package pushaction_test

import (
	"errors"
	"strings"

	"code.cloudfoundry.org/cli/actor/actionerror"
	. "code.cloudfoundry.org/cli/actor/pushaction"
	"code.cloudfoundry.org/cli/actor/pushaction/pushactionfakes"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/cli/util/manifest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes", func() {
	var (
		actor       *Actor
		fakeV2Actor *pushactionfakes.FakeV2Actor
	)

	BeforeEach(func() {
		fakeV2Actor = new(pushactionfakes.FakeV2Actor)
		actor = NewActor(fakeV2Actor, nil)
	})

	Describe("UnmapRoutes", func() {
		var (
			config ApplicationConfig

			returnedConfig ApplicationConfig
			warnings       Warnings
			executeErr     error
		)

		BeforeEach(func() {
			config = ApplicationConfig{
				DesiredApplication: Application{
					Application: v2action.Application{
						GUID: "some-app-guid",
					}},
			}
		})

		JustBeforeEach(func() {
			returnedConfig, warnings, executeErr = actor.UnmapRoutes(config)
		})

		Context("when there are routes on the application", func() {
			BeforeEach(func() {
				config.CurrentRoutes = []v2action.Route{
					{GUID: "some-route-guid-1", Host: "some-route-1", Domain: v2action.Domain{Name: "some-domain.com"}},
					{GUID: "some-route-guid-2", Host: "some-route-2"},
				}
			})

			Context("when the unmapping is successful", func() {
				BeforeEach(func() {
					fakeV2Actor.UnmapRouteFromApplicationReturns(v2action.Warnings{"unmap-route-warning"}, nil)
				})

				It("only creates the routes that do not exist", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("unmap-route-warning", "unmap-route-warning"))

					Expect(returnedConfig.CurrentRoutes).To(BeEmpty())

					Expect(fakeV2Actor.UnmapRouteFromApplicationCallCount()).To(Equal(2))

					routeGUID, appGUID := fakeV2Actor.UnmapRouteFromApplicationArgsForCall(0)
					Expect(routeGUID).To(Equal("some-route-guid-1"))
					Expect(appGUID).To(Equal("some-app-guid"))

					routeGUID, appGUID = fakeV2Actor.UnmapRouteFromApplicationArgsForCall(1)
					Expect(routeGUID).To(Equal("some-route-guid-2"))
					Expect(appGUID).To(Equal("some-app-guid"))
				})
			})

			Context("when the mapping errors", func() {
				var expectedErr error
				BeforeEach(func() {
					expectedErr = errors.New("oh my")
					fakeV2Actor.UnmapRouteFromApplicationReturns(v2action.Warnings{"unmap-route-warning"}, expectedErr)
				})

				It("sends the warnings and errors and returns true", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(warnings).To(ConsistOf("unmap-route-warning"))
				})
			})
		})
	})

	Describe("MapRoutes", func() {
		var (
			config ApplicationConfig

			returnedConfig ApplicationConfig
			boundRoutes    bool
			warnings       Warnings
			executeErr     error
		)

		BeforeEach(func() {
			config = ApplicationConfig{
				DesiredApplication: Application{
					Application: v2action.Application{
						GUID: "some-app-guid",
					}},
			}
		})

		JustBeforeEach(func() {
			returnedConfig, boundRoutes, warnings, executeErr = actor.MapRoutes(config)
		})

		Context("when routes need to be bound to the application", func() {
			BeforeEach(func() {
				config.CurrentRoutes = []v2action.Route{
					{GUID: "some-route-guid-2", Host: "some-route-2"},
				}
				config.DesiredRoutes = []v2action.Route{
					{GUID: "some-route-guid-1", Host: "some-route-1", Domain: v2action.Domain{Name: "some-domain.com"}},
					{GUID: "some-route-guid-2", Host: "some-route-2"},
					{GUID: "some-route-guid-3", Host: "some-route-3"},
				}
			})

			Context("when the mapping is successful", func() {
				BeforeEach(func() {
					fakeV2Actor.MapRouteToApplicationReturns(v2action.Warnings{"map-route-warning"}, nil)
				})

				It("only creates the routes that do not exist", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("map-route-warning", "map-route-warning"))
					Expect(boundRoutes).To(BeTrue())

					Expect(returnedConfig.CurrentRoutes).To(Equal(config.DesiredRoutes))

					Expect(fakeV2Actor.MapRouteToApplicationCallCount()).To(Equal(2))

					routeGUID, appGUID := fakeV2Actor.MapRouteToApplicationArgsForCall(0)
					Expect(routeGUID).To(Equal("some-route-guid-1"))
					Expect(appGUID).To(Equal("some-app-guid"))

					routeGUID, appGUID = fakeV2Actor.MapRouteToApplicationArgsForCall(1)
					Expect(routeGUID).To(Equal("some-route-guid-3"))
					Expect(appGUID).To(Equal("some-app-guid"))
				})
			})

			Context("when the mapping errors", func() {
				Context("when the route is bound in another space", func() {
					BeforeEach(func() {
						fakeV2Actor.MapRouteToApplicationReturns(v2action.Warnings{"map-route-warning"}, v2action.RouteInDifferentSpaceError{})
					})

					It("sends the RouteInDifferentSpaceError (with a guid set) and warnings and returns true", func() {
						Expect(executeErr).To(MatchError(v2action.RouteInDifferentSpaceError{Route: "some-route-1.some-domain.com"}))
						Expect(warnings).To(ConsistOf("map-route-warning"))
					})
				})

				Context("generic error", func() {
					var expectedErr error
					BeforeEach(func() {
						expectedErr = errors.New("oh my")
						fakeV2Actor.MapRouteToApplicationReturns(v2action.Warnings{"map-route-warning"}, expectedErr)
					})

					It("sends the warnings and errors and returns true", func() {
						Expect(executeErr).To(MatchError(expectedErr))
						Expect(warnings).To(ConsistOf("map-route-warning"))
					})
				})
			})
		})

		Context("when no routes need to be bound", func() {
			It("returns false", func() {
				Expect(executeErr).ToNot(HaveOccurred())
			})
		})
	})

	Describe("CalculateRoutes", func() {
		var (
			routes         []string
			orgGUID        string
			spaceGUID      string
			existingRoutes []v2action.Route

			calculatedRoutes []v2action.Route
			warnings         Warnings
			executeErr       error
		)

		BeforeEach(func() {
			routes = []string{
				"a.com",
				"b.a.com",
				"c.b.a.com",
				"d.c.b.a.com",
				"a.com/some-path",
			}
			orgGUID = "some-org-guid"
			spaceGUID = "some-space-guid"
		})

		JustBeforeEach(func() {
			calculatedRoutes, warnings, executeErr = actor.CalculateRoutes(routes, orgGUID, spaceGUID, existingRoutes)
		})

		Context("when there are no known routes", func() {
			BeforeEach(func() {
				existingRoutes = []v2action.Route{{
					GUID: "some-route-5",
					Host: "banana",
					Domain: v2action.Domain{
						GUID: "domain-guid-1",
						Name: "a.com",
					},
					SpaceGUID: spaceGUID,
				}}
			})

			Context("when a route looking up the domains is succuessful", func() {
				BeforeEach(func() {
					fakeV2Actor.GetDomainsByNameAndOrganizationReturns([]v2action.Domain{
						{GUID: "domain-guid-1", Name: "a.com"},
						{GUID: "domain-guid-2", Name: "b.a.com"},
					}, v2action.Warnings{"domain-warnings-1", "domains-warnings-2"}, nil)
				})

				Context("when the route existance check is successful", func() {
					BeforeEach(func() {
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"find-route-warning"}, v2action.RouteNotFoundError{})
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturnsOnCall(3, v2action.Route{
							GUID: "route-guid-4",
							Host: "d.c",
							Domain: v2action.Domain{
								GUID: "domain-guid-2",
								Name: "b.a.com",
							},
							SpaceGUID: spaceGUID,
						}, v2action.Warnings{"find-route-warning"}, nil)
					})

					It("returns new and existing routes", func() {
						Expect(executeErr).NotTo(HaveOccurred())
						Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2", "find-route-warning", "find-route-warning", "find-route-warning", "find-route-warning", "find-route-warning"))
						Expect(calculatedRoutes).To(ConsistOf(
							v2action.Route{
								Domain: v2action.Domain{
									GUID: "domain-guid-1",
									Name: "a.com",
								},
								SpaceGUID: spaceGUID,
							},
							v2action.Route{
								Domain: v2action.Domain{
									GUID: "domain-guid-2",
									Name: "b.a.com",
								},
								SpaceGUID: spaceGUID,
							},
							v2action.Route{
								Host: "c",
								Domain: v2action.Domain{
									GUID: "domain-guid-2",
									Name: "b.a.com",
								},
								SpaceGUID: spaceGUID,
							},
							v2action.Route{
								GUID: "route-guid-4",
								Host: "d.c",
								Domain: v2action.Domain{
									GUID: "domain-guid-2",
									Name: "b.a.com",
								},
								SpaceGUID: spaceGUID,
							},
							v2action.Route{
								GUID: "some-route-5",
								Host: "banana",
								Domain: v2action.Domain{
									GUID: "domain-guid-1",
									Name: "a.com",
								},
								SpaceGUID: spaceGUID,
							},
							v2action.Route{
								Host: "",
								Domain: v2action.Domain{
									GUID: "domain-guid-1",
									Name: "a.com",
								},
								Path:      "/some-path",
								SpaceGUID: spaceGUID,
							}))

						Expect(fakeV2Actor.GetDomainsByNameAndOrganizationCallCount()).To(Equal(1))
						domains, passedOrgGUID := fakeV2Actor.GetDomainsByNameAndOrganizationArgsForCall(0)
						Expect(domains).To(ConsistOf("a.com", "b.a.com", "c.b.a.com", "d.c.b.a.com"))
						Expect(passedOrgGUID).To(Equal(orgGUID))

						Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsCallCount()).To(Equal(5))
						// One check is enough here - checking 4th call since it's the only
						// existing one.
						Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsArgsForCall(3)).To(Equal(v2action.Route{
							Host: "d.c",
							Domain: v2action.Domain{
								GUID: "domain-guid-2",
								Name: "b.a.com",
							},
							SpaceGUID: spaceGUID,
						}))
					})
				})

				Context("when the route existance check fails", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("oh noes")
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"find-route-warning"}, expectedErr)
					})

					It("returns back warnings and error", func() {
						Expect(executeErr).To(MatchError(expectedErr))
						Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2", "find-route-warning"))
					})
				})

				Context("when one of the domains does not exist", func() {
					BeforeEach(func() {
						fakeV2Actor.GetDomainsByNameAndOrganizationReturns(nil, v2action.Warnings{"domain-warnings-1", "domains-warnings-2"}, nil)
					})

					It("returns back warnings and error", func() {
						Expect(executeErr).To(MatchError(actionerror.NoMatchingDomainError{Route: "a.com"}))
						Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2"))
					})
				})

				Context("when TCP properties are being set on a HTTP domain", func() {
					BeforeEach(func() {
						routes = []string{"a.com", "b.a.com", "c.b.a.com:1234"}
					})

					It("returns back warnings and error", func() {
						Expect(executeErr).To(MatchError(actionerror.InvalidHTTPRouteSettings{Domain: "b.a.com"}))
						Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2"))
					})
				})
			})

			Context("when looking up a domain returns an error", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("po-tate-toe")
					fakeV2Actor.GetDomainsByNameAndOrganizationReturns(nil, v2action.Warnings{"domain-warnings-1", "domains-warnings-2"}, expectedErr)
				})

				It("returns back warnings and error", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2"))
				})
			})
		})

		Context("when there are known routes", func() {
			BeforeEach(func() {
				existingRoutes = []v2action.Route{{
					GUID: "route-guid-4",
					Host: "d.c",
					Domain: v2action.Domain{
						GUID: "domain-guid-2",
						Name: "b.a.com",
					},
					SpaceGUID: spaceGUID,
				}}

				fakeV2Actor.GetDomainsByNameAndOrganizationReturns([]v2action.Domain{
					{GUID: "domain-guid-1", Name: "a.com"},
					{GUID: "domain-guid-2", Name: "b.a.com"},
				}, v2action.Warnings{"domain-warnings-1", "domains-warnings-2"}, nil)
				fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"find-route-warning"}, v2action.RouteNotFoundError{})
			})

			It("does not lookup known routes", func() {
				Expect(executeErr).NotTo(HaveOccurred())
				Expect(warnings).To(ConsistOf("domain-warnings-1", "domains-warnings-2", "find-route-warning", "find-route-warning", "find-route-warning", "find-route-warning"))
				Expect(calculatedRoutes).To(ConsistOf(
					v2action.Route{
						Domain: v2action.Domain{
							GUID: "domain-guid-1",
							Name: "a.com",
						},
						SpaceGUID: spaceGUID,
					},
					v2action.Route{
						Domain: v2action.Domain{
							GUID: "domain-guid-2",
							Name: "b.a.com",
						},
						SpaceGUID: spaceGUID,
					},
					v2action.Route{
						Host: "c",
						Domain: v2action.Domain{
							GUID: "domain-guid-2",
							Name: "b.a.com",
						},
						SpaceGUID: spaceGUID,
					},
					v2action.Route{
						GUID: "route-guid-4",
						Host: "d.c",
						Domain: v2action.Domain{
							GUID: "domain-guid-2",
							Name: "b.a.com",
						},
						SpaceGUID: spaceGUID,
					},
					v2action.Route{
						Host: "",
						Domain: v2action.Domain{
							GUID: "domain-guid-1",
							Name: "a.com",
						},
						Path:      "/some-path",
						SpaceGUID: spaceGUID,
					}))

				Expect(fakeV2Actor.GetDomainsByNameAndOrganizationCallCount()).To(Equal(1))
				domains, passedOrgGUID := fakeV2Actor.GetDomainsByNameAndOrganizationArgsForCall(0)
				Expect(domains).To(ConsistOf("a.com", "b.a.com", "c.b.a.com"))
				Expect(passedOrgGUID).To(Equal(orgGUID))
			})
		})
	})

	Describe("CreateAndMapDefaultApplicationRoute", func() {
		var (
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			warnings, executeErr = actor.CreateAndMapDefaultApplicationRoute("some-org-guid", "some-space-guid",
				v2action.Application{Name: "some-app", GUID: "some-app-guid"})
		})

		Context("when getting organization domains errors", func() {
			BeforeEach(func() {
				fakeV2Actor.GetOrganizationDomainsReturns(
					[]v2action.Domain{},
					v2action.Warnings{"domain-warning"},
					errors.New("some-error"))
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError("some-error"))
				Expect(warnings).To(ConsistOf("domain-warning"))
			})
		})

		Context("when getting organization domains succeeds", func() {
			BeforeEach(func() {
				fakeV2Actor.GetOrganizationDomainsReturns(
					[]v2action.Domain{
						{
							GUID: "some-domain-guid",
							Name: "some-domain",
						},
					},
					v2action.Warnings{"domain-warning"},
					nil,
				)
			})

			Context("when getting the application routes errors", func() {
				BeforeEach(func() {
					fakeV2Actor.GetApplicationRoutesReturns(
						[]v2action.Route{},
						v2action.Warnings{"route-warning"},
						errors.New("some-error"),
					)
				})

				It("returns the error", func() {
					Expect(executeErr).To(MatchError("some-error"))
					Expect(warnings).To(ConsistOf("domain-warning", "route-warning"))
				})
			})

			Context("when getting the application routes succeeds", func() {
				// TODO: do we need this context
				Context("when the route is already bound to the app", func() {
					BeforeEach(func() {
						fakeV2Actor.GetApplicationRoutesReturns(
							[]v2action.Route{
								{
									Host: "some-app",
									Domain: v2action.Domain{
										GUID: "some-domain-guid",
										Name: "some-domain",
									},
									GUID:      "some-route-guid",
									SpaceGUID: "some-space-guid",
								},
							},
							v2action.Warnings{"route-warning"},
							nil,
						)
					})

					It("returns any warnings", func() {
						Expect(executeErr).ToNot(HaveOccurred())
						Expect(warnings).To(ConsistOf("domain-warning", "route-warning"))

						Expect(fakeV2Actor.GetOrganizationDomainsCallCount()).To(Equal(1), "Expected GetOrganizationDomains to be called once, but it was not")
						orgGUID := fakeV2Actor.GetOrganizationDomainsArgsForCall(0)
						Expect(orgGUID).To(Equal("some-org-guid"))

						Expect(fakeV2Actor.GetApplicationRoutesCallCount()).To(Equal(1), "Expected GetApplicationRoutes to be called once, but it was not")
						appGUID := fakeV2Actor.GetApplicationRoutesArgsForCall(0)
						Expect(appGUID).To(Equal("some-app-guid"))

						Expect(fakeV2Actor.CreateRouteCallCount()).To(Equal(0), "Expected CreateRoute to not be called but it was")
						Expect(fakeV2Actor.MapRouteToApplicationCallCount()).To(Equal(0), "Expected MapRouteToApplication to not be called but it was")
					})
				})

				Context("when the route isn't bound to the app", func() {
					Context("when finding route in space errors", func() {
						BeforeEach(func() {
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(
								v2action.Route{},
								v2action.Warnings{"route-warning"},
								errors.New("some-error"),
							)
						})

						It("returns the error", func() {
							Expect(executeErr).To(MatchError("some-error"))
							Expect(warnings).To(ConsistOf("domain-warning", "route-warning"))
						})
					})

					Context("when the route exists", func() {
						BeforeEach(func() {
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(
								v2action.Route{
									GUID: "some-route-guid",
									Host: "some-app",
									Domain: v2action.Domain{
										Name: "some-domain",
										GUID: "some-domain-guid",
									},
									SpaceGUID: "some-space-guid",
								},
								v2action.Warnings{"route-warning"},
								nil,
							)
						})

						Context("when the map command returns an error", func() {
							BeforeEach(func() {
								fakeV2Actor.MapRouteToApplicationReturns(
									v2action.Warnings{"map-warning"},
									errors.New("some-error"),
								)
							})

							It("returns the error", func() {
								Expect(executeErr).To(MatchError("some-error"))
								Expect(warnings).To(ConsistOf("domain-warning", "route-warning", "map-warning"))
							})
						})

						Context("when the map command succeeds", func() {
							BeforeEach(func() {
								fakeV2Actor.MapRouteToApplicationReturns(
									v2action.Warnings{"map-warning"},
									nil,
								)
							})

							It("maps the route to the app and returns any warnings", func() {
								Expect(executeErr).ToNot(HaveOccurred())
								Expect(warnings).To(ConsistOf("domain-warning", "route-warning", "map-warning"))

								Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsCallCount()).To(Equal(1), "Expected FindRouteBoundToSpaceWithSettings to be called once, but it was not")
								spaceRoute := fakeV2Actor.FindRouteBoundToSpaceWithSettingsArgsForCall(0)
								Expect(spaceRoute).To(Equal(v2action.Route{
									Host: "some-app",
									Domain: v2action.Domain{
										Name: "some-domain",
										GUID: "some-domain-guid",
									},
									SpaceGUID: "some-space-guid",
								}))

								Expect(fakeV2Actor.MapRouteToApplicationCallCount()).To(Equal(1), "Expected MapRouteToApplication to be called once, but it was not")
								routeGUID, appGUID := fakeV2Actor.MapRouteToApplicationArgsForCall(0)
								Expect(routeGUID).To(Equal("some-route-guid"))
								Expect(appGUID).To(Equal("some-app-guid"))
							})
						})
					})

					Context("when the route does not exist", func() {
						BeforeEach(func() {
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(
								v2action.Route{},
								v2action.Warnings{"route-warning"},
								v2action.RouteNotFoundError{},
							)
						})

						Context("when the create route command errors", func() {
							BeforeEach(func() {
								fakeV2Actor.CreateRouteReturns(
									v2action.Route{},
									v2action.Warnings{"route-create-warning"},
									errors.New("some-error"),
								)
							})

							It("returns the error", func() {
								Expect(executeErr).To(MatchError("some-error"))
								Expect(warnings).To(ConsistOf("domain-warning", "route-warning", "route-create-warning"))
							})
						})

						Context("when the create route command succeeds", func() {
							BeforeEach(func() {
								fakeV2Actor.CreateRouteReturns(
									v2action.Route{
										GUID: "some-route-guid",
										Host: "some-app",
										Domain: v2action.Domain{
											Name: "some-domain",
											GUID: "some-domain-guid",
										},
										SpaceGUID: "some-space-guid",
									},
									v2action.Warnings{"route-create-warning"},
									nil,
								)
							})

							Context("when the map command errors", func() {
								BeforeEach(func() {
									fakeV2Actor.MapRouteToApplicationReturns(
										v2action.Warnings{"map-warning"},
										errors.New("some-error"),
									)
								})

								It("returns the error", func() {
									Expect(executeErr).To(MatchError("some-error"))
									Expect(warnings).To(ConsistOf("domain-warning", "route-warning", "route-create-warning", "map-warning"))
								})
							})
							Context("when the map command succeeds", func() {

								BeforeEach(func() {
									fakeV2Actor.MapRouteToApplicationReturns(
										v2action.Warnings{"map-warning"},
										nil,
									)
								})

								It("creates the route, maps it to the app, and returns any warnings", func() {
									Expect(executeErr).ToNot(HaveOccurred())
									Expect(warnings).To(ConsistOf("domain-warning", "route-warning", "route-create-warning", "map-warning"))

									Expect(fakeV2Actor.CreateRouteCallCount()).To(Equal(1), "Expected CreateRoute to be called once, but it was not")
									defaultRoute, shouldGeneratePort := fakeV2Actor.CreateRouteArgsForCall(0)
									Expect(defaultRoute).To(Equal(v2action.Route{
										Host: "some-app",
										Domain: v2action.Domain{
											Name: "some-domain",
											GUID: "some-domain-guid",
										},
										SpaceGUID: "some-space-guid",
									}))
									Expect(shouldGeneratePort).To(BeFalse())

									Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsCallCount()).To(Equal(1), "Expected FindRouteBoundToSpaceWithSettings to be called once, but it was not")
									spaceRoute := fakeV2Actor.FindRouteBoundToSpaceWithSettingsArgsForCall(0)
									Expect(spaceRoute).To(Equal(v2action.Route{
										Host: "some-app",
										Domain: v2action.Domain{
											Name: "some-domain",
											GUID: "some-domain-guid",
										},
										SpaceGUID: "some-space-guid",
									}))

									Expect(fakeV2Actor.MapRouteToApplicationCallCount()).To(Equal(1), "Expected MapRouteToApplication to be called once, but it was not")
									routeGUID, appGUID := fakeV2Actor.MapRouteToApplicationArgsForCall(0)
									Expect(routeGUID).To(Equal("some-route-guid"))
									Expect(appGUID).To(Equal("some-app-guid"))
								})
							})
						})
					})
				})
			})
		})
	})

	Describe("CreateRoutes", func() {
		var (
			config ApplicationConfig

			returnedConfig ApplicationConfig
			createdRoutes  bool
			warnings       Warnings
			executeErr     error
		)

		BeforeEach(func() {
			config = ApplicationConfig{}
		})

		JustBeforeEach(func() {
			returnedConfig, createdRoutes, warnings, executeErr = actor.CreateRoutes(config)
		})

		Describe("when routes need to be created", func() {
			BeforeEach(func() {
				config.DesiredRoutes = []v2action.Route{
					{GUID: "", Host: "some-route-1"},
					{GUID: "some-route-guid-2", Host: "some-route-2"},
					{GUID: "", Host: "some-route-3"},
					{GUID: "", Host: "", Domain: v2action.Domain{RouterGroupType: constant.TCPRouterGroup}},
				}
			})

			Context("when the creation is successful", func() {
				BeforeEach(func() {
					fakeV2Actor.CreateRouteReturnsOnCall(0, v2action.Route{GUID: "some-route-guid-1", Host: "some-route-1"}, v2action.Warnings{"create-route-warning"}, nil)
					fakeV2Actor.CreateRouteReturnsOnCall(1, v2action.Route{GUID: "some-route-guid-3", Host: "some-route-3"}, v2action.Warnings{"create-route-warning"}, nil)
					fakeV2Actor.CreateRouteReturnsOnCall(2, v2action.Route{GUID: "some-route-guid-4", Domain: v2action.Domain{RouterGroupType: constant.TCPRouterGroup}}, v2action.Warnings{"create-route-warning"}, nil)
				})

				It("only creates the routes that do not exist", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("create-route-warning", "create-route-warning", "create-route-warning"))
					Expect(createdRoutes).To(BeTrue())
					Expect(returnedConfig.DesiredRoutes).To(Equal([]v2action.Route{
						{GUID: "some-route-guid-1", Host: "some-route-1"},
						{GUID: "some-route-guid-2", Host: "some-route-2"},
						{GUID: "some-route-guid-3", Host: "some-route-3"},
						{GUID: "some-route-guid-4", Domain: v2action.Domain{RouterGroupType: constant.TCPRouterGroup}},
					}))

					Expect(fakeV2Actor.CreateRouteCallCount()).To(Equal(3))

					passedRoute, randomRoute := fakeV2Actor.CreateRouteArgsForCall(0)
					Expect(passedRoute).To(Equal(v2action.Route{Host: "some-route-1"}))
					Expect(randomRoute).To(BeFalse())

					passedRoute, randomRoute = fakeV2Actor.CreateRouteArgsForCall(1)
					Expect(passedRoute).To(Equal(v2action.Route{Host: "some-route-3"}))
					Expect(randomRoute).To(BeFalse())

					passedRoute, randomRoute = fakeV2Actor.CreateRouteArgsForCall(2)
					Expect(passedRoute).To(Equal(v2action.Route{GUID: "", Host: "", Domain: v2action.Domain{RouterGroupType: constant.TCPRouterGroup}}))
					Expect(randomRoute).To(BeTrue())
				})
			})

			Context("when the creation errors", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("oh my")
					fakeV2Actor.CreateRouteReturns(
						v2action.Route{},
						v2action.Warnings{"create-route-warning"},
						expectedErr)
				})

				It("sends the warnings and errors and returns true", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(warnings).To(ConsistOf("create-route-warning"))
				})
			})
		})

		Context("when no routes are created", func() {
			BeforeEach(func() {
				config.DesiredRoutes = []v2action.Route{
					{GUID: "some-route-guid-1", Host: "some-route-1"},
					{GUID: "some-route-guid-2", Host: "some-route-2"},
					{GUID: "some-route-guid-3", Host: "some-route-3"},
				}
			})

			It("returns false", func() {
				Expect(createdRoutes).To(BeFalse())
			})
		})
	})

	Describe("GetGeneratedRoute", func() {
		var (
			providedManifest manifest.Application
			orgGUID          string
			spaceGUID        string
			knownRoutes      []v2action.Route

			defaultRoute v2action.Route
			warnings     Warnings
			executeErr   error

			domain v2action.Domain
		)

		BeforeEach(func() {
			providedManifest = manifest.Application{
				Name: "Some-App",
			}
			orgGUID = "some-org-guid"
			spaceGUID = "some-space-guid"
			knownRoutes = nil

			domain = v2action.Domain{
				Name: "private-domain.com",
				GUID: "some-private-domain-guid",
			}
		})

		JustBeforeEach(func() {
			defaultRoute, warnings, executeErr = actor.GetGeneratedRoute(providedManifest, orgGUID, spaceGUID, knownRoutes)
		})

		Context("the domain is provided", func() {
			BeforeEach(func() {
				providedManifest.Domain = "some-private-domain"
			})

			Context("when the provided domain exists", func() {
				Context("when the provided domain is an HTTP domain", func() {
					BeforeEach(func() {
						fakeV2Actor.GetDomainsByNameAndOrganizationReturns(
							[]v2action.Domain{domain},
							v2action.Warnings{"some-organization-domain-warning"},
							nil,
						)
					})

					Context("when the route does not exist", func() {
						BeforeEach(func() {
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"get-route-warnings"}, v2action.RouteNotFoundError{})
						})

						It("it uses the provided domain instead of the first shared domain", func() {
							Expect(executeErr).ToNot(HaveOccurred())
							Expect(warnings).To(ConsistOf("some-organization-domain-warning", "get-route-warnings"))
							Expect(defaultRoute).To(Equal(v2action.Route{
								Domain:    domain,
								Host:      strings.ToLower(providedManifest.Name),
								SpaceGUID: spaceGUID,
							}))

							Expect(fakeV2Actor.GetDomainsByNameAndOrganizationCallCount()).To(Equal(1))
							domainNamesArg, orgGUIDArg := fakeV2Actor.GetDomainsByNameAndOrganizationArgsForCall(0)
							Expect(domainNamesArg).To(Equal([]string{"some-private-domain"}))
							Expect(orgGUIDArg).To(Equal(orgGUID))
						})
					})

					Context("when the route exists in the current space", func() {
						var route v2action.Route

						BeforeEach(func() {
							route = v2action.Route{
								Domain:    domain,
								GUID:      "some-route-guid",
								Host:      strings.ToLower(providedManifest.Name),
								SpaceGUID: spaceGUID,
							}
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(route, v2action.Warnings{"get-route-warnings"}, nil)
						})

						It("returns the route and warnings", func() {
							Expect(executeErr).ToNot(HaveOccurred())
							Expect(warnings).To(ConsistOf("some-organization-domain-warning", "get-route-warnings"))
							Expect(defaultRoute).To(Equal(route))
						})
					})

					Context("when finding the route errors", func() {
						var expectedErr error

						BeforeEach(func() {
							expectedErr = errors.New("some crazy stuff")
							fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"get-route-warnings"}, expectedErr)
						})

						It("returns the warnings and errors", func() {
							Expect(executeErr).To(MatchError(expectedErr))
							Expect(warnings).To(ConsistOf("some-organization-domain-warning", "get-route-warnings"))
						})
					})
				})

				Context("when the provided domain is an TCP domain", func() {
					BeforeEach(func() {
						domain.RouterGroupType = constant.TCPRouterGroup

						fakeV2Actor.GetDomainsByNameAndOrganizationReturns(
							[]v2action.Domain{domain},
							v2action.Warnings{"some-organization-domain-warning"},
							nil,
						)

						// Assumes new route
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"get-route-warnings"}, v2action.RouteNotFoundError{})
					})

					It("it uses the provided domain instead of the first shared domain and has no host", func() {
						Expect(executeErr).ToNot(HaveOccurred())
						Expect(warnings).To(ConsistOf("some-organization-domain-warning", "get-route-warnings"))
						Expect(defaultRoute).To(Equal(v2action.Route{
							Domain:    domain,
							SpaceGUID: spaceGUID,
						}))

						Expect(fakeV2Actor.GetDomainsByNameAndOrganizationCallCount()).To(Equal(1))
						domainNamesArg, orgGUIDArg := fakeV2Actor.GetDomainsByNameAndOrganizationArgsForCall(0)
						Expect(domainNamesArg).To(Equal([]string{"some-private-domain"}))
						Expect(orgGUIDArg).To(Equal(orgGUID))
					})
				})
			})

			Context("when the provided domain does not exist", func() {
				BeforeEach(func() {
					fakeV2Actor.GetDomainsByNameAndOrganizationReturns(
						[]v2action.Domain{},
						v2action.Warnings{"some-organization-domain-warning"},
						nil,
					)
				})

				It("returns an DomainNotFoundError", func() {
					Expect(executeErr).To(MatchError(v2action.DomainNotFoundError{Name: "some-private-domain"}))
					Expect(warnings).To(ConsistOf("some-organization-domain-warning"))
				})
			})
		})

		Context("when the domain is not provided", func() {
			Context("when retrieving the domains is successful", func() {
				BeforeEach(func() {
					fakeV2Actor.GetOrganizationDomainsReturns(
						[]v2action.Domain{domain},
						v2action.Warnings{"private-domain-warnings", "shared-domain-warnings"},
						nil,
					)
				})

				Context("when the route exists", func() {
					BeforeEach(func() {
						// Assumes new route
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{
							Domain:    domain,
							GUID:      "some-route-guid",
							Host:      strings.ToLower(providedManifest.Name),
							SpaceGUID: spaceGUID,
						}, v2action.Warnings{"get-route-warnings"}, nil)
					})

					It("returns the route and warnings", func() {
						Expect(executeErr).ToNot(HaveOccurred())
						Expect(warnings).To(ConsistOf("private-domain-warnings", "shared-domain-warnings", "get-route-warnings"))

						Expect(defaultRoute).To(Equal(v2action.Route{
							Domain:    domain,
							GUID:      "some-route-guid",
							Host:      strings.ToLower(providedManifest.Name),
							SpaceGUID: spaceGUID,
						}))

						Expect(fakeV2Actor.GetOrganizationDomainsCallCount()).To(Equal(1))
						Expect(fakeV2Actor.GetOrganizationDomainsArgsForCall(0)).To(Equal(orgGUID))

						Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsCallCount()).To(Equal(1))
						Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsArgsForCall(0)).To(Equal(v2action.Route{Domain: domain, Host: strings.ToLower(providedManifest.Name), SpaceGUID: spaceGUID}))
					})

					Context("when the route has been found", func() {
						BeforeEach(func() {
							knownRoutes = []v2action.Route{{
								Domain:    domain,
								GUID:      "some-route-guid",
								Host:      strings.ToLower(providedManifest.Name),
								SpaceGUID: spaceGUID,
							}}
						})

						It("should return the known route and warnings", func() {
							Expect(executeErr).ToNot(HaveOccurred())
							Expect(warnings).To(ConsistOf("private-domain-warnings", "shared-domain-warnings"))

							Expect(defaultRoute).To(Equal(v2action.Route{
								Domain:    domain,
								GUID:      "some-route-guid",
								Host:      strings.ToLower(providedManifest.Name),
								SpaceGUID: spaceGUID,
							}))

							Expect(fakeV2Actor.FindRouteBoundToSpaceWithSettingsCallCount()).To(Equal(0))
						})
					})
				})

				Context("when the route does not exist", func() {
					BeforeEach(func() {
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"get-route-warnings"}, v2action.RouteNotFoundError{})
					})

					It("returns a partial route", func() {
						Expect(executeErr).ToNot(HaveOccurred())
						Expect(warnings).To(ConsistOf("private-domain-warnings", "shared-domain-warnings", "get-route-warnings"))

						Expect(defaultRoute).To(Equal(v2action.Route{Domain: domain, Host: strings.ToLower(providedManifest.Name), SpaceGUID: spaceGUID}))
					})
				})

				Context("when retrieving the routes errors", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("whoops")
						fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(v2action.Route{}, v2action.Warnings{"get-route-warnings"}, expectedErr)
					})

					It("returns errors and warnings", func() {
						Expect(executeErr).To(MatchError(expectedErr))
						Expect(warnings).To(ConsistOf("private-domain-warnings", "shared-domain-warnings", "get-route-warnings"))
					})
				})
			})

			Context("when retrieving the domains errors", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("whoops")
					fakeV2Actor.GetOrganizationDomainsReturns([]v2action.Domain{}, v2action.Warnings{"private-domain-warnings", "shared-domain-warnings"}, expectedErr)
				})

				It("returns errors and warnings", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(warnings).To(ConsistOf("private-domain-warnings", "shared-domain-warnings"))
				})
			})
		})
	})
})
