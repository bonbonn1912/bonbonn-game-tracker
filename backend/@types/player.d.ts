import faceitElo, {faceitEloSchema} from "./level"

export default interface faceitPlayerReponse {
    local: faceitElo,
    player_id: string,
    nickname: string,
    avatar: string,
    country: string,
    cover_image: string,
    platforms: platform,
    games: games,
}

interface platform {
    steam?: string
}

interface games {
    csgo: csgo
}

interface csgo {
    region: string,
    game_player_id: string,
    skill_level: number,
    faceit_elo: number,
    game_player_name: string,
}

