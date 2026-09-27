// 백엔드가 남의 Gemini·KIS 호출을 대신 쳐 주는 공짜 프록시가 되지 않도록,
// 모든 요청에 공유 암호를 실어 보낸다. 사람마다 계정을 두는 게 아니라 암호 하나를
// 아는 사람만 들어오는 구조다.
export const TOKEN_HEADER = 'X-App-Token'

const KEY = 'onefolio-token'

// 암호는 IndexedDB(자산 데이터) 가 아니라 localStorage 에 둔다. 자산 기록을
// 지우는 초기화가 암호까지 날려서 다시 입력하게 만들 이유가 없다.
export function loadToken(): string | null {
  return localStorage.getItem(KEY)
}

export function saveToken(token: string): void {
  localStorage.setItem(KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(KEY)
}
