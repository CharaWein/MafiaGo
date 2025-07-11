package game

const (
	RoleDon      = "don"
	RoleMafia    = "mafia"
	RoleSheriff  = "sheriff"
	RoleCivilian = "civilian"
)

type Role struct {
	Name        string
	Description string
	NightAction bool
}

var Roles = map[string]Role{
	RoleDon: {
		Name:        "Дон мафии",
		Description: "Может проверять игроков на роль шерифа",
		NightAction: true,
	},
	RoleMafia: {
		Name:        "Мафия",
		Description: "Участвует в ночных убийствах",
		NightAction: true,
	},
	RoleSheriff: {
		Name:        "Шериф",
		Description: "Может проверять игроков на принадлежность к мафии",
		NightAction: true,
	},
	RoleCivilian: {
		Name:        "Мирный житель",
		Description: "Выжить и вычислить мафию",
		NightAction: false,
	},
}
