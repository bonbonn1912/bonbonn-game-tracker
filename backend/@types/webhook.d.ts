import { type event } from './event'

export default interface webHookBody {
  transaction_id: string
  event: event
  event_id: string
  third_party_id: string
  app_id: string
  timestamp: string
  retry_count: number
  version: number
  payload: payload
  map?: string // from /matches/{matchId}
  matchup: matchup
  streamer: string,
}

interface matchup {
   team1 : firstTeam,
   team2: secondTeam
}

interface firstTeam{
    name: string,
    avgElo: number,
}

interface secondTeam{
    name: string,
    avgElo: number,
}

interface payload {
  id: string
  organizer_id: string
  region: string
  game: 'csgo'
  version: number
  entity: entity
  teams: team[]
  created_at: string
  updated_at: string
  started_at: string
  finished_at: string
}

interface entity {
  id: string
  name: string
  type: string
}

interface team {
  id: string
  name: string
  type: string
  avatar: string
  leader_id: string
  co_leader_id: string
  roster: player[]
}

interface player {
  id: string
  nickname: string
  avatar: string
  game_id: string
  game_name: string
  game_skill_level: number
  membership: string
  anticheat_required: boolean
}

export type { payload, entity, team, player, matchup, firstTeam, secondTeam }
