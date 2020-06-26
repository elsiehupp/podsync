package feed

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	itunes "github.com/eduncan911/podcast"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mxpv/podsync/pkg/config"
	"github.com/mxpv/podsync/pkg/model"
)

// sort.Interface implementation
type timeSlice []*model.Episode

func (p timeSlice) Len() int {
	return len(p)
}

// In descending order
func (p timeSlice) Less(i, j int) bool {
	return p[i].PubDate.After(p[j].PubDate)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func Build(ctx context.Context, feed *model.Feed, cfg *config.Feed, provider urlProvider) (*itunes.Podcast, error) {
	const (
		podsyncGenerator = "Podsync generator (support us at https://github.com/mxpv/podsync)"
		// defaultCategory  = "TV & Film"
	)

	var (
		now = time.Now().UTC()
	)

	p := itunes.New(feed.Title, feed.ItemURL, feed.Description, &feed.PubDate, &now)
	p.Generator = podsyncGenerator

	itunesCategories := map[string]bool{
		"Arts":                    true,
		"Business":                true,
		"Comedy":                  true,
		"Education":               true,
		"Fiction":                 true,
		"Government":              true,
		"History":                 true,
		"Health & Fitness":        true,
		"Kids & Family":           true,
		"Leisure":                 true,
		"Music":                   true,
		"News":                    true,
		"Religion & Spirituality": true,
		"Science":                 true,
		"Society & Culture":       true,
		"Sports":                  true,
		"Technology":              true,
		"True Crime":              true,
		"TV & Film":               true,
	}

	itunesSubcategories := map[string]string{
		"Books":            "Arts",
		"Design":           "Arts",
		"Fashion & Beauty": "Arts",
		"Food":             "Arts",
		"Performing Arts":  "Arts",
		"Visual Arts":      "Arts",

		"Careers":          "Business",
		"Entrepreneurship": "Business",
		"Investing":        "Business",
		"Management":       "Business",
		"Marketing":        "Business",
		"Non-Profit":       "Business",

		"Comedy Interviews": "Comedy",
		"Improv":            "Comedy",
		"Stand-Up":          "Comedy",

		"Courses":           "Education",
		"How To":            "Education",
		"Language Learning": "Education",
		"Self-Improvement":  "Education",

		"Comedy Fiction":  "Fiction",
		"Drama":           "Fiction",
		"Science Fiction": "Fiction",

		"Alternative Health": "Health & Fitness",
		"Fitness":            "Health & Fitness",
		"Medicine":           "Health & Fitness",
		"Mental Health":      "Health & Fitness",
		"Nutrition":          "Health & Fitness",
		"Sexuality":          "Health & Fitness",

		"Education for Kids": "Kids & Family",
		"Parenting":          "Kids & Family",
		"Pets & Animals":     "Kids & Family",
		"Stories for Kids":   "Kids & Family",

		"Animation & Manga": "Leisure",
		"Automotive":        "Leisure",
		"Aviation":          "Leisure",
		"Crafts":            "Leisure",
		"Games":             "Leisure",
		"Hobbies":           "Leisure",
		"Home & Garden":     "Leisure",
		"Video Games":       "Leisure",

		"Music Commentary": "Music",
		"Music History":    "Music",
		"Music Interviews": "Music",

		"Business News":      "News",
		"Daily News":         "News",
		"Entertainment News": "News",
		"News Commentary":    "News",
		"Politics":           "News",
		"Sports News":        "News",
		"Tech News":          "News",

		"Buddhism":     "Religion & Spirituality",
		"Christianity": "Religion & Spirituality",
		"Hinduism":     "Religion & Spirituality",
		"Islam":        "Religion & Spirituality",
		"Judaism":      "Religion & Spirituality",
		"Religion":     "Religion & Spirituality",
		"Spirituality": "Religion & Spirituality",

		"Astronomy":        "Science",
		"Chemistry":        "Science",
		"Earth Sciences":   "Science",
		"Life Sciences":    "Science",
		"Mathematics":      "Science",
		"Natural Sciences": "Science",
		"Nature":           "Science",
		"Physics":          "Science",
		"Social Sciences":  "Science",

		"Documentary":       "Society & Culture",
		"Personal Journals": "Society & Culture",
		"Philosophy":        "Society & Culture",
		"Places & Travel":   "Society & Culture",
		"Relationships":     "Society & Culture",

		"Baseball":       "Sports",
		"Basketball":     "Sports",
		"Cricket":        "Sports",
		"Fantasy Sports": "Sports",
		"Football":       "Sports",
		"Golf":           "Sports",
		"Hockey":         "Sports",
		"Rugby":          "Sports",
		"Running":        "Sports",
		"Soccer":         "Sports",
		"Swimming":       "Sports",
		"Tennis":         "Sports",
		"Volleyball":     "Sports",
		"Wilderness":     "Sports",
		"Wrestling":      "Sports",

		"After Shows":     "TV & Film",
		"Film History":    "TV & Film",
		"Film Interviews": "TV & Film",
		"Film Reviews":    "TV & Film",
		"TV Reviews":      "TV & Film",
	}

	var warningCounter = 0

	if cfg.Metadata.Title != "" {
		p.Title = cfg.Metadata.Title
	} else {
		p.Title = feed.Title
		warningCounter++
		log.Warn("No feed title specified. Using feed title from source.")
	}

	if cfg.Metadata.Description != "" {
		p.AddSummary(cfg.Metadata.Description)
	} else {
		p.AddSummary(feed.Description)
		warningCounter++
		log.Warn("No feed description specified. Using feed description from source.")
	}

	if cfg.Metadata.CoverArt != "" {
		p.AddImage(cfg.Metadata.CoverArt)
	} else {
		p.AddImage(feed.CoverArt)
		warningCounter++
		log.Warn("No feed image specified. Using feed image from source.")
	}

	if cfg.Metadata.Subtitle != "" {
		p.ISubtitle = cfg.Metadata.Subtitle
	} else {
		warningCounter++
		log.Warn("No feed subtitle specified.")
	}

	if cfg.Metadata.Author != "" {
		p.IAuthor = cfg.Metadata.Author
	} else {
		warningCounter++
		log.Warn("No feed author specified.")
	}

	if cfg.Metadata.Language != "" {
		p.Language = cfg.Metadata.Language
	} else {
		warningCounter++
		log.Warn("No feed language specified. Feed language left blank.")
	}

	for key := range itunesCategories {
		if key == cfg.Metadata.Category {
			break
		} else {
			warningCounter++
			log.Warn("Category does not match list of Apple Podcasts categories")
		}
	}

	for key, value := range itunesSubcategories {
		if key == cfg.Metadata.Subcategory {
			if value == cfg.Metadata.Category {
				break
			} else {
				warningCounter++
				log.Warn("Category does not correspond to parent category")
				break
			}
		} else {
			warningCounter++
			log.Warn("Subcategory does not match list of Apple Podcasts categories")
		}
	}

	if cfg.Metadata.Category != "" {
		if cfg.Metadata.Subcategory != "" {
			var subcategory []string
			subcategory[0] = cfg.Metadata.Subcategory
			p.AddCategory(cfg.Metadata.Category, subcategory)
		} else {
			p.AddCategory(cfg.Metadata.Category, nil)
			warningCounter++
			log.Warn("No feed subcategory specified. Feed subcategory left blank.")
		}
	} else {
		warningCounter++
		log.Warn("No feed category specified. Feed category left blank.")
		if cfg.Metadata.Subcategory != "" {
			var subcategory []string
			subcategory[0] = cfg.Metadata.Subcategory
			p.AddCategory("", subcategory)
		} else {
			warningCounter++
			log.Warn("No feed subcategory specified. Feed subcategory left blank.")
		}
	}

	if cfg.Metadata.AdminName != "" {
		if cfg.Metadata.AdminEmail != "" {
			p.AddAuthor(cfg.Metadata.AdminName, cfg.Metadata.AdminEmail)
		} else {
			warningCounter++
			log.Warn("No feed admin email specified. Feed admin email left blank.")
			p.AddAuthor(cfg.Metadata.AdminName, "")
		}
	} else {
		warningCounter++
		log.Warn("No feed admin name specified. Feed admin name left blank.")
		if cfg.Metadata.AdminEmail != "" {
			p.AddAuthor("", cfg.Metadata.AdminEmail)
		} else {
			warningCounter++
			log.Warn("No feed admin email specified. Feed admin email left blank.")
		}
	}

	if cfg.Metadata.Explicit == "true" || cfg.Metadata.Explicit == "yes" || cfg.Metadata.Explicit == "explicit" {
		p.IExplicit = "yes"
	} else if cfg.Metadata.Explicit == "false" || cfg.Metadata.Explicit == "no" || cfg.Metadata.Explicit == "clean" {
		p.IExplicit = "no"
	} else {
		warningCounter++
		log.Warn("No (or invalid) explicit tag specified.")
	}

	// Insert Copyright
	if cfg.Metadata.Copyright != "" {
		p.Copyright = cfg.Metadata.Copyright
	} else {
		warningCounter++
		log.Warn("No copyright string tag specified. Copyright tag left blank.")

	}

	// Insert Allow/Block
	if cfg.Metadata.AllowItunes == "true" || cfg.Metadata.AllowItunes == "yes" || cfg.Metadata.AllowItunes == "disallow" {
		if warningCounter > 0 {
			log.Warn("Podcast feed does not meet Apple Podcasts specifications and may not appear in Apple Podcasts.")
		}
	} else {
		log.Warn("Podcast will explicitly disallow Apple Podcasts from indexing it.")
		p.IBlock = "yes"
	}

	for _, episode := range feed.Episodes {
		if episode.PubDate.IsZero() {
			episode.PubDate = now
		}
	}

	// Sort all episodes in descending order
	sort.Sort(timeSlice(feed.Episodes))

	for i, episode := range feed.Episodes {
		if episode.Status != model.EpisodeDownloaded {
			// Skip episodes that are not yet downloaded
			continue
		}

		item := itunes.Item{
			GUID:        episode.ID,
			Link:        episode.VideoURL,
			Title:       episode.Title,
			Description: episode.Description,
			ISubtitle:   episode.Title,
			// Some app prefer 1-based order
			IOrder: strconv.Itoa(i + 1),
		}

		item.AddPubDate(&episode.PubDate)
		item.AddSummary(episode.Description)
		item.AddImage(episode.Thumbnail)
		item.AddDuration(episode.Duration)

		enclosureType := itunes.MP4
		if feed.Format == model.FormatAudio {
			enclosureType = itunes.MP3
		}

		episodeName := EpisodeName(cfg, episode)
		downloadURL, err := provider.URL(ctx, cfg.ID, episodeName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to obtain download URL for: %s", episodeName)
		}

		item.AddEnclosure(downloadURL, enclosureType, episode.Size)

		// p.AddItem requires description to be not empty, use workaround
		if item.Description == "" {
			item.Description = " "
		}

		if p.IExplicit == "yes" {
			item.IExplicit = "yes"
		}

		if _, err := p.AddItem(item); err != nil {
			return nil, errors.Wrapf(err, "failed to add item to podcast (id %q)", episode.ID)
		}
	}

	return &p, nil
}

func EpisodeName(feedConfig *config.Feed, episode *model.Episode) string {
	ext := "mp4"
	if feedConfig.Format == model.FormatAudio {
		ext = "mp3"
	}

	return fmt.Sprintf("%s.%s", episode.ID, ext)
}
