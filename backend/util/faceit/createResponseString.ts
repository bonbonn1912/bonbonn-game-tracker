import faceitElo from "../../@types/level"

const eloReponse = (rankInformation: faceitElo) : string =>{
    if(rankInformation.isMax){
        return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo`
    }else{
        return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo. Elo needed for level ${rankInformation.level +1}: ${rankInformation.eloToNextLevel} `
    }
}

export { eloReponse }