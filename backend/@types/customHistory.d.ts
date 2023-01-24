interface singlePlayer {
    player_id: string,
    nickname: string,
    avatar: string,
    skill_level: number,
    game_player_id: string,
    game_player_name: string,
    faceit_url: string,
}

interface faction1 {
    team_id: string,
    nickname: string,
    avatar: string,
    type?: string,
    players: singlePlayer[]
}

interface faction2 {
    team_id: string,
    nickname: string,
    avatar: string,
    type?: string,
    players: players
}

interface results {
    winner: string,
    score: score
}

interface score {
    faction1: number,
    facetion2: number
}

interface teams {
    faction1: faction1,
    faction2: faction2,
}

interface matchHistory {
    items: items[]

}

interface items{
    teams: teams,
    results: results
}

export { matchHistory, faction1, faction2, singlePlayer }
