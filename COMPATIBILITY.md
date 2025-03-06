# Compatiblity
At present moment, we have only recreated the v1 api, which spans from 2006-2012. Out of this span, only 2010 to 2012 works due to basic authentication

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

⭐✅: 4.1.3, Near perfect. Tweeting with media attached requires the BlueTweety tweak, Retweets made by you fail to appear properly on timeline, pagination broken on some minor elements. Notification settings crashes the app.

⭐✅: 5.0.0-5.0.3, same as 4.1.3. 5.0.2+ has the aspect ratio change implemented for the iPhone 5.

❌: 5.0.4+ Uses Twitter API v1.1

### Twitter iOS Integration

✅: iOS 5-6, Works through Bluetweety

❌: iOS 7+, Uses Twitter API v1.1

### Tweetie2

### Tweetie
❌: Uses basic authentication

## Android
⭐⚠️: 3.3.0, Partly works, Followers & following timelines do not work, Connect tab is missing follows & retweets.

⚠️: 3.1.2, same as 3.3.0
