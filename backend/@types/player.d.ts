import type faceitElo from './level'
import { faceitEloSchema } from './level'
import { type InsertType } from '../util/database/Schema/insertTypes'

export default interface faceitPlayerReponse {
  insertType?: InsertType
  local: faceitElo
  player_id: string
  nickname: string
  avatar: string
  country: string
  cover_image: string
  platforms: platform
  games: games
  matchHistory?: string[]
}

interface platform {
  steam?: string
}

interface games {
  csgo: csgo
  cs2?: cs2
}

interface csgo {
  region: string
  game_player_id: string
  skill_level: number
  faceit_elo: number
  game_player_name: string
}

interface cs2 {
  region: string
  game_player_id: string
  skill_level: number
  faceit_elo: number
  game_player_name: string
}

interface faceitMatchHistory {
  lifetime: lifetime
}

interface lifetime {
  'Recent Results': string[]
}

export type { faceitMatchHistory }
