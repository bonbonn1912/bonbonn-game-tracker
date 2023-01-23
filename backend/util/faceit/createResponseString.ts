import faceitElo from "../../@types/level"

const eloReponse = (rankInformation: faceitElo) : string =>{
    if(rankInformation.isMax){
        return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo`
    }else{
        return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo. Elo needed for level ${rankInformation.level +1}: ${rankInformation.eloToNextLevel} `
    }
}

const appendHistory = (matchHistory: string[]) : string =>{
    if(matchHistory.length == 0){
        return ""
    }
    let winLose = matchHistory.reverse().map(result => result == "1" ? "W/" : "L/")
    let winLoseString = ".Last 5 Games: "+winLose.join("")
    return winLoseString.substring(0, winLoseString.length-1)

}

export { eloReponse, appendHistory }