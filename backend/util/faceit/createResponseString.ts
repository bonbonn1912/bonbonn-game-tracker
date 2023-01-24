import faceitElo from '../../@types/level'

const eloReponse = (rankInformation: faceitElo) : string => {
  if (rankInformation.isMax) {
    return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo`
  } else {
    return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo. Elo needed for level ${rankInformation.level + 1}: ${rankInformation.eloToNextLevel} `
  }
}

const appendHistory = (matchHistory: string[], firstPart: string) : string => {
  if (matchHistory.length == 0) {
    return ''
  }
  const winLose = matchHistory.reverse().map(result => result == '1' ? 'W/' : 'L/')
  const winLoseString = firstPart + winLose.join('')
  return winLoseString.substring(0, winLoseString.length - 1)
}

const appendWinRate = (matchHistory: string[]) => {
  if (matchHistory.length == 0) {
    return ''
  }
  const arraySum: number = matchHistory.map(el => parseInt(el)).reduce((curr: number, prev: number) => curr + prev)
  const winrate: number = (arraySum / matchHistory.length)
  return `. Winrate: ${Math.floor(winrate * 100)}%`
}

export { eloReponse, appendHistory, appendWinRate }
