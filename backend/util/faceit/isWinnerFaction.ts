import { type faction1, type faction2, type singlePlayer } from '../../@types/customHistory'

const isWinnerFaction = (player_id: string, winnerFaction: faction1 | faction2) => {
  let winner: boolean = false
  if (isInWinnerFaction(player_id, winnerFaction.players)) {
    winner = true
  }

  return winner
}

const isInWinnerFaction = (player_id: string, players: singlePlayer[]) => {
  let isInFaction: boolean = false
  players.forEach(singlePlayer => {
    if (player_id == singlePlayer.player_id) {
      isInFaction = true
    }
  })
  return isInFaction
}

export { isWinnerFaction }
