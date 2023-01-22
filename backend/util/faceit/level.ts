import faceitElo from "../../types/level";

const faceitRankInformation = (elo: number): faceitElo => {
  let baseElo: number = elo - 800;
  let baseLevel: number = 1;
  let rankInformation: faceitElo = {
    isMax: false,
    level: 0,
    elo: elo,
    eloToNextLevel: 0,
  };
  if (baseElo <= 0) {
    rankInformation.isMax = false;
    rankInformation.elo = elo;
    rankInformation.level = baseLevel;
    rankInformation.eloToNextLevel = eloNeeded(elo, baseLevel);
  } else {
    let level = (baseElo - 1) / 150;
    if (level >= 8) {
      rankInformation.isMax = true;
      rankInformation.elo = elo;
      rankInformation.level = 10;
    } else {
      rankInformation.isMax = false;
      rankInformation.elo = elo;
      rankInformation.level = Math.floor(level) + 2;
      rankInformation.eloToNextLevel = eloNeeded(elo, Math.floor(level) + 2);
    }
  }
  return rankInformation;
};

const minEloForLevel: Array<number> = [
  1, 801, 951, 1101, 1251, 1401, 1551, 1701, 1851, 2001,
];

const eloNeeded = (elo: number, level: number): number => {
  return minEloForLevel[level] - elo;
};

export {faceitRankInformation}
