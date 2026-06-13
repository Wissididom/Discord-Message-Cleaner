async function fetchWithRateLimitHandling(url: string, options: RequestInit) {
  let response = await fetch(url, options);
  if (response.status === 429) {
    const retryAfter = parseInt(response.headers.get("Retry-After") || "0", 10);
    if (retryAfter > 0) {
      // https://docs.discord.com/developers/topics/rate-limits#exceeding-a-rate-limit
      console.log(
        `${
          new Date().toLocaleString()
        } - Rate limit hit. Retrying after ${retryAfter}s...`,
      );
      await new Promise((resolve) => setTimeout(resolve, retryAfter * 1000)); // wait for the retry period
      response = await fetchWithRateLimitHandling(url, options); // retry the request after waiting
    }
  }
  return response;
}

async function getMessages(
  discordToken: string,
  channelId: string,
  before: string | undefined = undefined,
  after: string | undefined = undefined,
  limit: number = 100,
): Promise<
  { id: string; authorName: string; created: string; content: string }[]
> {
  let url = `https://discord.com/api/v10/channels/${channelId}/messages`;
  const params = [];
  if (before) {
    params.push(`before=${before}`);
  }
  if (after) {
    params.push(`after=${after}`);
  }
  params.push(`limit=${limit}`);
  if (params.length > 0) {
    url += `?${params.join("&")}`;
  }
  const messagesResp = await fetchWithRateLimitHandling(url, {
    method: "GET",
    headers: {
      Authorization: `Bot ${discordToken}`,
      "User-Agent": "OldMessageDeletor (wissididom.de, 1)",
    },
  });
  if (messagesResp.ok) {
    // deno-lint-ignore no-explicit-any
    const messages = await messagesResp.json() as any[];
    return messages.map((message) => {
      return {
        id: message.id,
        authorName: message.author.username,
        created: message.timestamp,
        content: message.content,
      };
    });
  } else {
    throw `getMessages: got ${messagesResp.status} ${messagesResp.statusText}`;
  }
}

async function deleteMessage(
  discordToken: string,
  channelId: string,
  msgId: string,
  reason: string | undefined = undefined,
): Promise<boolean> {
  const deleteResp = await fetchWithRateLimitHandling(
    `https://discord.com/api/v10/channels/${channelId}/messages/${msgId}`,
    {
      method: "DELETE",
      headers: {
        Authorization: `Bot ${discordToken}`,
        "User-Agent": "OldMessageDeletor (wissididom.de, 1)",
        "X-Audit-Log-Reason": reason ?? "Pepe-Deletor",
      },
    },
  );
  return deleteResp.ok;
}

const discordToken = Deno.env.get("DISCORD_TOKEN");
const guildId = Deno.env.get("SERVER_ID");
const channelId = Deno.env.get("CHANNEL_ID");
if (discordToken && guildId && channelId) {
  let after = undefined;
  if (Deno.env.get("START_WITH_OLDEST")?.toLowerCase() == "true") {
    after = "1";
  }
  const messages = await getMessages(
    discordToken,
    channelId,
    undefined, //before
    after,
  );
  if (after == "1") messages.reverse(); // Start with oldest
  for (const message of messages) {
    console.log(
      `${
        new Date(
          message.created,
        ).toLocaleString()
      } - ${message.authorName}: ${message.content ? message.content : "N/A"}`,
    );
  }
  const doDelete = confirm("Do you want to delete the above messages?");
  let i = 1;
  for (const message of messages) {
    if (doDelete) {
      console.log(
        `[${i}/100] Deleting ${message.id} ${
          new Date(
            message.created,
          ).toLocaleString()
        } - ${message.authorName}: ${
          message.content ? message.content : "N/A"
        }`,
      );
      await deleteMessage(discordToken, channelId, message.id);
      console.log(
        `[${i}/100] Deleted ${message.id} ${
          new Date(
            message.created,
          ).toLocaleString()
        } - ${message.authorName}: ${
          message.content ? message.content : "N/A"
        }`,
      );
      i++;
    }
  }
}

export {}; // Make this file a module so top-level-await works
