import { faction1, faction2, singlePlayer } from '../../@types/customHistory'

const isWinnerFaction = (player_id: string, winnerFaction: faction1 | faction2) => {
  let winner: Boolean = false
  if (isInWinnerFaction(player_id, winnerFaction.players)) {
    winner = true
  }

  return winner
}

const isInWinnerFaction = (player_id: string, players: singlePlayer[]) => {
  let isInFaction: Boolean = false
  players.forEach(singlePlayer => {
    if (player_id == singlePlayer.player_id) {
      isInFaction = true
    }
  })
  return isInFaction
}

export { isWinnerFaction }
