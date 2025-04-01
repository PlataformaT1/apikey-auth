package server

func (s *Server) ApiKey(router *Router) {

	router.AddRoute("GET", "/apikey/validate", s.apikeyHandler.HandleValidateApiKey)

}
