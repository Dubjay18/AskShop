package contracts

import "strings"

// JoinPaths joins URL path segments ensuring exactly one slash between parts.
func JoinPaths(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	res := parts[0]
	for i := 1; i < len(parts); i++ {
		left := strings.TrimRight(res, "/")
		right := strings.TrimLeft(parts[i], "/")
		res = left + "/" + right
	}
	if res == "" {
		return "/"
	}
	return res
}

const (
	APIV1      = "/api/v1"
	ServiceAPI = "/api" // for internal services that don't version externally
)

type AuthRouteSet struct {
	Base           string
	Register       string
	Login          string
	Refresh        string
	Logout         string
	Profile        string // GET/PUT same path
	ChangePassword string
	// Relative suffixes (useful when creating a Gin group at Base)
	SuffixProfile        string
	SuffixChangePassword string
}

type CRUDRouteSet struct {
	Base string
	ByID string
	// Relative suffixes
	SuffixByID string
}

type GatewayRouteSet struct {
	APIBase  string
	Auth     AuthRouteSet
	Products CRUDRouteSet
	Users    CRUDRouteSet
}

var Routes = GatewayRouteSet{
	APIBase: APIV1,
	Auth: AuthRouteSet{
		Base:                 JoinPaths(APIV1, "/auth"),
		Register:             JoinPaths(APIV1, "/auth", "/register"),
		Login:                JoinPaths(APIV1, "/auth", "/login"),
		Refresh:              JoinPaths(APIV1, "/auth", "/refresh"),
		Logout:               JoinPaths(APIV1, "/auth", "/logout"),
		Profile:              JoinPaths(APIV1, "/auth", "/profile"),
		ChangePassword:       JoinPaths(APIV1, "/auth", "/change-password"),
		SuffixProfile:        "/profile",
		SuffixChangePassword: "/change-password",
	},
	Products: CRUDRouteSet{
		Base:       JoinPaths(APIV1, "/products"),
		ByID:       JoinPaths(APIV1, "/products", "/:id"),
		SuffixByID: "/:id",
	},
	Users: CRUDRouteSet{
		Base:       JoinPaths(APIV1, "/users"),
		ByID:       JoinPaths(APIV1, "/users", "/:id"),
		SuffixByID: "/:id",
	},
}
