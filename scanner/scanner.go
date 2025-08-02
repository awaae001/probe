package scanner

import (
	"context"
	"discord-bot/database"
	"discord-bot/models"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
)

const maxPartitionConcurrency = 45          // 每个服务器内最大并发分区扫描数
const maxThreadConcurrencyPerPartition = 24 // 每个分区内最大并发线程处理数

// StartScanning initiates the concurrent scanning process.
func StartScanning(s *discordgo.Session, scanningConfig models.ScanningConfig, isFullScan bool) {
	log.Println("Starting the scanning process...")

	if len(scanningConfig) == 0 {
		log.Println("No valid guild configurations found.")
		return
	}

	var totalPartitions int
	var tasks []models.PartitionTask

	for guildID, guildConfig := range scanningConfig {
		log.Printf("Preparing to scan guild: %s (%s)", guildConfig.Name, guildID)

		if err := database.InitDB(guildConfig.DBPath); err != nil {
			log.Printf("Failed to initialize database for guild %s: %v", guildID, err)
			continue
		}

		for key, channelConfig := range guildConfig.Data {
			if len(channelConfig.ChannelID) > 0 {
				// Scan only specified channels
				for _, chID := range channelConfig.ChannelID {
					task := models.PartitionTask{
						DB:          database.DB,
						GuildConfig: &guildConfig,
						ChannelID:   chID,
						Key:         key,
						IsFullScan:  isFullScan,
					}
					tasks = append(tasks, task)
					totalPartitions++
				}
			} else {
				// Scan all forum channels in the category if no specific channels are listed
				channels, err := s.GuildChannels(guildID)
				if err != nil {
					log.Printf("Failed to get channels for guild %s: %v", guildID, err)
					continue
				}
				for _, channel := range channels {
					if channel.Type == discordgo.ChannelTypeGuildForum && channel.ParentID == channelConfig.ID {
						task := models.PartitionTask{
							DB:          database.DB,
							GuildConfig: &guildConfig,
							ChannelID:   channel.ID,
							Key:         key,
							IsFullScan:  isFullScan,
						}
						tasks = append(tasks, task)
						totalPartitions++
					}
				}
			}
		}
	}

	if totalPartitions == 0 {
		log.Println("No partitions to scan.")
		return
	}

	var partitionsDone int64
	var totalNewPostsFound int64

	taskChan := make(chan models.PartitionTask, totalPartitions)
	for _, task := range tasks {
		task.TotalPartitions = totalPartitions
		task.PartitionsDone = &partitionsDone
		task.TotalNewPostsFound = &totalNewPostsFound
		taskChan <- task
	}
	close(taskChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	numWorkers := min(totalPartitions, maxPartitionConcurrency)
	log.Printf("Starting %d workers for %d partitions.", numWorkers, totalPartitions)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i+1, s, ctx, taskChan, &wg)
	}

	wg.Wait()
	log.Printf("Scanning process finished. Found %d new posts in total.", atomic.LoadInt64(&totalNewPostsFound))
}

// worker is the core processing unit in the pool.
func worker(id int, s *discordgo.Session, ctx context.Context, tasks <-chan models.PartitionTask, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		defer atomic.AddInt64(task.PartitionsDone, 1)

		select {
		case <-ctx.Done():
			log.Printf("Worker %d cancelling task for partition %s.", id, task.Key)
			return
		default:
		}

		startTime := time.Now()
		channelID := task.ChannelID
		tableName := "channel_" + channelID

		if err := database.CreateTableForChannel(task.DB, tableName); err != nil {
			log.Printf("Error creating table %s: %v", tableName, err)
			return
		}

		existingThreads := make(map[string]bool)
		existingThreadsMutex := &sync.RWMutex{}

		if !task.IsFullScan {
			postIDs, err := database.GetAllPostIDs(task.DB, tableName)
			if err != nil {
				log.Printf("Error getting all post IDs for active scan from table %s: %v", tableName, err)
			} else {
				existingThreads = postIDs
			}
		}

		excludedThreads, err := database.GetExcludedThreads(task.DB, task.GuildConfig.GuildsID, channelID)
		if err != nil {
			log.Printf("Error getting excluded threads for channel %s: %v", channelID, err)
		} else {
			for threadID := range excludedThreads {
				existingThreads[threadID] = true
			}
		}

		processThreadsConcurrently := func(threads []*discordgo.Channel, threadType string) {
			log.Printf("Processing %d %s threads for channel %s", len(threads), threadType, channelID)
			if len(threads) == 0 {
				return
			}

			completedPartitions := atomic.LoadInt64(task.PartitionsDone)
			remainingPartitions := task.TotalPartitions - int(completedPartitions)
			optimalThreadsPerPartition := calculateOptimalThreadsPerPartition(remainingPartitions)

			chunkSize := (len(threads) + optimalThreadsPerPartition - 1) / optimalThreadsPerPartition
			if chunkSize == 0 {
				chunkSize = 1
			}

			chunks := chunkThreads(threads, chunkSize)
			var chunkWg sync.WaitGroup
			semaphore := make(chan struct{}, optimalThreadsPerPartition)

			for _, chunk := range chunks {
				chunkWg.Add(1)
				go func(c models.ThreadChunk) {
					defer chunkWg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					processThreadsChunk(s, c, existingThreads, existingThreadsMutex, task, tableName, ctx)
				}(chunk)
			}
			chunkWg.Wait()
		}

		activeThreads, err := s.ThreadsActive(channelID)
		if err != nil {
			log.Printf("Error getting active threads for channel %s: %v", channelID, err)
			return
		}
		processThreadsConcurrently(activeThreads.Threads, "active")

		if task.IsFullScan {
			var before *time.Time
			pageCount := 0
			for {
				pageCount++
				select {
				case <-ctx.Done():
					log.Println("Scan cancelled during pagination.")
					return
				default:
				}

				archivedThreads, err := s.ThreadsArchived(channelID, before, 100)
				if err != nil {
					log.Printf("Error getting archived threads for channel %s on page %d: %v", channelID, pageCount, err)
					break
				}

				log.Printf("Page %d: Fetched %d archived threads for channel %s. HasMore: %v", pageCount, len(archivedThreads.Threads), channelID, archivedThreads.HasMore)

				if len(archivedThreads.Threads) == 0 {
					log.Printf("No more archived threads found for channel %s on page %d.", channelID, pageCount)
					break
				}

				processThreadsConcurrently(archivedThreads.Threads, "archived")

				if !archivedThreads.HasMore {
					log.Printf("Stopping archived thread fetch for channel %s: HasMore is false.", channelID)
					break
				}

				lastThread := archivedThreads.Threads[len(archivedThreads.Threads)-1]
				if lastThread.ThreadMetadata == nil {
					log.Printf("Archived thread %s has no metadata, stopping pagination.", lastThread.ID)
					break
				}
				before = &lastThread.ThreadMetadata.ArchiveTimestamp
			}
		}
		log.Printf("Partition %s (%s) scan completed in %v", task.Key, task.GuildConfig.Name, time.Since(startTime))
	}
}

func processThreadsChunk(s *discordgo.Session, chunk models.ThreadChunk, existingThreads map[string]bool, existingThreadsMutex *sync.RWMutex, task models.PartitionTask, tableName string, ctx context.Context) {
	for _, thread := range chunk.Threads {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if thread.ThreadMetadata != nil && thread.ThreadMetadata.Locked {
			continue
		}

		existingThreadsMutex.RLock()
		_, exists := existingThreads[thread.ID]
		existingThreadsMutex.RUnlock()
		if exists {
			// log.Printf("Skipping thread %s, already exists in database.", thread.ID)
			continue
		}

		firstMessage, err := s.ChannelMessage(thread.ID, thread.ID)
		if err != nil {
			if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Response.StatusCode == 404 {
				log.Printf("Thread %s not found (404), adding to exclusion list.", thread.ID)
				if err := database.AddThreadToExclusionList(task.DB, task.GuildConfig.GuildsID, task.ChannelID, thread.ID, "Not Found"); err != nil {
					log.Printf("Error adding thread %s to exclusion list: %v", thread.ID, err)
				}
			} else {
				log.Printf("Error getting first message for thread %s: %v", thread.ID, err)
			}
			continue
		}

		var tagNames []string
		if thread.AppliedTags != nil {
			for _, tagID := range thread.AppliedTags {
				tagNames = append(tagNames, string(tagID))
			}
		}

		content := firstMessage.Content
		runes := []rune(content)
		if len(runes) > 512 {
			content = string(runes[:512])
		}

		var coverImageURL string
		if len(firstMessage.Attachments) > 0 {
			coverImageURL = firstMessage.Attachments[0].URL
		}

		totalReactions := 0
		uniqueUserIDs := make(map[string]struct{})

		for _, reaction := range firstMessage.Reactions {
			totalReactions += reaction.Count
			users, err := s.MessageReactions(thread.ID, firstMessage.ID, reaction.Emoji.APIName(), 100, "", "")
			if err != nil {
				log.Printf("Error getting users for reaction %s on message %s: %v", reaction.Emoji.APIName(), firstMessage.ID, err)
				continue
			}
			for _, user := range users {
				uniqueUserIDs[user.ID] = struct{}{}
			}
		}
		uniqueReactions := len(uniqueUserIDs)

		post := models.Post{
			ThreadID:        thread.ID,
			ChannelID:       thread.ParentID,
			Title:           thread.Name,
			Author:          firstMessage.Author.Username,
			AuthorID:        firstMessage.Author.ID,
			Content:         content,
			Tags:            strings.Join(tagNames, ","),
			MessageCount:    thread.MessageCount,
			Timestamp:       firstMessage.Timestamp.Unix(),
			CoverImageURL:   coverImageURL,
			TotalReactions:  totalReactions,
			UniqueReactions: uniqueReactions,
		}

		if err := database.InsertPost(task.DB, post, tableName); err != nil {
			log.Printf("Error inserting post %s into database: %v", post.ThreadID, err)
		} else {
			atomic.AddInt64(task.TotalNewPostsFound, 1)
			existingThreadsMutex.Lock()
			existingThreads[post.ThreadID] = true
			existingThreadsMutex.Unlock()
			log.Printf("Successfully saved post: %s to table %s", post.ThreadID, tableName)
		}
	}
}

func chunkThreads(threads []*discordgo.Channel, chunkSize int) []models.ThreadChunk {
	var chunks []models.ThreadChunk
	for i := 0; i < len(threads); i += chunkSize {
		end := min(i+chunkSize, len(threads))
		chunks = append(chunks, models.ThreadChunk{
			Threads: threads[i:end],
			Index:   len(chunks),
		})
	}
	return chunks
}

func calculateOptimalThreadsPerPartition(remainingPartitions int) int {
	if remainingPartitions <= 0 {
		return maxThreadConcurrencyPerPartition
	}

	const minThreadsPerPartition = 4
	var threadsPerPartition int

	if remainingPartitions <= 3 {
		threadsPerPartition = maxThreadConcurrencyPerPartition
	} else if remainingPartitions <= 5 {
		threadsPerPartition = 12
	} else if remainingPartitions <= 10 {
		threadsPerPartition = 8
	} else {
		threadsPerPartition = minThreadsPerPartition
	}

	return min(threadsPerPartition, maxThreadConcurrencyPerPartition)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
