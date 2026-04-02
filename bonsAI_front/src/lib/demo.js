const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export const demoSensors = {
  temperature: 24.6,
  humidity: 58,
  soilMoisture: 43,
  illuminance: 12800,
  lastUpdated: "just now",
  source: "fallback"
};

export async function streamDemoReply({ message, sensors, onDelta }) {
  const lowerMessage = message.toLowerCase();
  const needsWater =
    typeof sensors.soilMoisture === "number" && sensors.soilMoisture < 35;
  const brightEnough =
    typeof sensors.illuminance === "number" && sensors.illuminance > 9000;

  let response =
    "こんにちは。盆栽の今の気配を見ながら、一緒に様子を整理してみます。";

  if (lowerMessage.includes("水") || lowerMessage.includes("乾")) {
    response = needsWater
      ? "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。"
      : "土壌水分はまだ保たれています。今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。";
  } else if (lowerMessage.includes("元気") || lowerMessage.includes("状態")) {
    response = brightEnough
      ? "照度は確保できていて、盆栽は比較的落ち着いた環境にいます。温湿度と土壌水分のバランスも大きく崩れていません。"
      : "強い異常は見えませんが、やや光量が控えめです。置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。";
  } else if (lowerMessage.includes("ありがとう")) {
    response =
      "どういたしまして。小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。";
  } else {
    response = `${response} 現在は${sensors.temperature}℃、湿度${sensors.humidity}%、土壌水分${sensors.soilMoisture}%、照度${sensors.illuminance}lxです。質問があれば、水やり・日照・体調の見立てに寄せて返答できます。`;
  }

  const chunks = response.split(/(?<=[。 ]|、)/).filter(Boolean);

  for (const chunk of chunks) {
    await wait(130);
    onDelta(chunk);
  }
}
