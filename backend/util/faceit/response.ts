import faceitElo from "../../types/level"

const eloReponse = (username: string, rankInformation: faceitElo) : string =>{
    if(rankInformation.isMax){
        return `${username} is level ${rankInformation.level} with ${rankInformation.elo} elo`
    }else{
        return `${username} is level ${rankInformation.level} with ${rankInformation.elo} elo. Elo needed for level ${rankInformation.level +1}: ${rankInformation.eloToNextLevel} `
    }
}

export { eloReponse }