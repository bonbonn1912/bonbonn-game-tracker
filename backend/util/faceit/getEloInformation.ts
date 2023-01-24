import type faceitElo from '../../@types/level'
import type faceitPlayerReponse from '../../@types/player'

const faceitRankInformation = (player: faceitPlayerReponse): faceitElo => {
  const baseElo: number = player.games.csgo.faceit_elo - 800
  const elo = player.games.csgo.faceit_elo
  const baseLevel: number = 1
  const rankInformation: faceitElo = {
    responseString: '',
    username: player.nickname,
    isMax: false,
    level: 0,
    elo,
    eloToNextLevel: 0
  }
  if (baseElo <= 0) {
    rankInformation.isMax = false
    rankInformation.elo = elo
    rankInformation.level = baseLevel
    rankInformation.eloToNextLevel = eloNeeded(elo, baseLevel)
  } else {
    const level = (baseElo - 1) / 150
    if (level >= 8) {
      rankInformation.isMax = true
      rankInformation.elo = elo
      rankInformation.level = 10
    } else {
      rankInformation.isMax = false
      rankInformation.elo = elo
      rankInformation.level = Math.floor(level) + 2
      rankInformation.eloToNextLevel = eloNeeded(player.games.csgo.faceit_elo, Math.floor(level) + 2)
    }
  }
  return rankInformation
}

const minEloForLevel: number[] = [
  1, 801, 951, 1101, 1251, 1401, 1551, 1701, 1851, 2001
]

const eloNeeded = (elo: number, level: number): number => {
  return minEloForLevel[level] - elo
}

export { faceitRankInformation }
