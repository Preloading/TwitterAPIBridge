# Compatiblity
At present moment, we have only recreated the v1 api, which spans from 2006-2012.

Twitter API v2 is unlikely to be fixed in the future.

## Key
 Key| Meaning |
|---| ------- |
| ⭐ | Actively testing on |
| ✅ | Works |
| ❓ | Untested
| ⚠️ | Partly works (only use this for features which aren't implemented on a specific platform) |
| ❌ | Doesn't work |
| 💾 | Lost |
| 🔒 | Won't Fix |

## iOS

### Twitter for iOS (offical)

⭐⚠️: 3.3.6, Uses a **lot** of XML, Posts with images are broken (have (null) at the start), many other faults and missing endpoints

⭐✅: 4.1.3, Near perfect. Tweeting media requires a patch, Retweets made by you fail to appear properly on timeline, pagination broken on some user elements. Notification settings crashes the app.

⚠️: 5.0.0-5.1.2, Images fail to load properly

❌: 5.2+ Uses Twitter API v1.1

### Twitter iOS Integration

❓: iOS 5, needs patch, untested

✅: iOS 6, tested thru proxy, needs patch

### Tweetie2

### Tweetie
💾: lost

## Android
⭐⚠️: 3.3.0, Partly works, Followers & following timelines do not work, Connect tab is missing follows & retweets.

⚠️: 3.1.2, same as 3.3.0
