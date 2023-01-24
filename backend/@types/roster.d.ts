export default interface teamFromMatch {
    name: string,
    avatar: string,
    roster: rosterPLayer[]
}

interface rosterPLayer {
    id: string,
    nickname: string,
}

export { rosterPLayer }