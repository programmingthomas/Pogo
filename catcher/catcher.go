package catcher

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/programmingthomas/Pogo/pogoutils"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//A podcast episode (as stored in pogoconfig.json)
type PodEpisode struct {
	URL                           string
	Description                   template.HTML
	ShouldDownloadIfNotDownloaded bool
	Title                         string
	Author                        string
	Summary                       string
	PubDate                       string
	Type                          string
	Length                        time.Duration
	Image                         string
}

//A podcast feed (as stored in pogoconfig.json)
type PodFeed struct {
	Name            string
	FeedURL         string
	Site            string
	LastRefreshed   time.Time
	Language        string
	PodcastEpisodes []PodEpisode
	Copyright       string
	Subtitle        string
	Description     string
	Summary         string
	Image           string
	Categories      []string
	ID              string
	Acronym         string
}

//A catcher is the tool that will catch the podcasts and run a scheduled loop in the
//background
type Catcher struct {
	Podcasts        []PodFeed
	RefreshInterval time.Duration
	ConfigLocation  string
	addFeed         chan PodFeed
	ticker          *time.Ticker
	updatedPodcasts chan []PodFeed
}

//Open a catcher from the given file (creating it if it doesn't exist) and start catching
//podcasts
func StartCatcher(configSaveLocation string) Catcher {
	if !pogoutils.FileExists("downloads/") {
		pogoutils.CreateFolder("downloads")
	}
	catcher := Catcher{}
	_, er := os.Stat(configSaveLocation)
	if er == nil {
		contents, err := ioutil.ReadFile(configSaveLocation)
		if err == nil {
			json.Unmarshal(contents, &catcher)
			fmt.Println("Loaded from file")
		}
	} else {
		//Initial creation of a catcher
		catcher.ConfigLocation = configSaveLocation
		catcher.RefreshInterval = time.Minute * 30
		catcher.SaveData()
	}
	//Temporary default for debugging purposes...
	catcher.RefreshInterval = time.Minute * 2
	catcher.addFeed = make(chan PodFeed)
	catcher.updatedPodcasts = make(chan []PodFeed)
	catcher.ticker = time.NewTicker(catcher.RefreshInterval)
	go catcher.Refresher()
	return catcher
}

//A concurrent task that will refresh podcasts
func (catcher *Catcher) Refresher() {
	//Refresh once at the beginning
	catcher.RefreshAllPodcasts()
	for {
		select {
		case <-catcher.ticker.C:
			//Ticker fired
			catcher.RefreshAllPodcasts()
		case u := <-catcher.addFeed:
			//Received a new feed; add it
			fmt.Println("Adding", u.Name)
			catcher.Podcasts = append(catcher.Podcasts, u)
			u.Refresh(catcher)
			catcher.updatedPodcasts <- catcher.Podcasts
			fmt.Println("But I got here!!!")
		}
	}
}

//Should be run concurrently to refresh all podcasts
func (catcher *Catcher) RefreshAllPodcasts() {
	fmt.Println("Refreshing all podcasts", len(catcher.Podcasts))
	for _, podcast := range catcher.Podcasts {
		podcast.Refresh(catcher)
	}
	catcher.updatedPodcasts <- catcher.Podcasts
	go catcher.SaveData()
}

//Refresh an individual podcast
func (podFeed *PodFeed) Refresh(parent *Catcher) {
	fmt.Println("Refreshing", podFeed.Name)
	resp, err := http.Get(podFeed.FeedURL)
	if err == nil {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			var xmlResponse Fetched
			err := xml.Unmarshal(contents, &xmlResponse)
			if err == nil {
				podcast := parent.getPodcastFromXML(xmlResponse, podFeed.FeedURL)
				for _, episode := range podcast.PodcastEpisodes {
					added := false
					for _, existingEpisode := range podFeed.PodcastEpisodes {
						if existingEpisode.URL == episode.URL {
							added = true
							break
						}
					}
					if !added {
						fmt.Println("Added", episode.URL)
						episode.ShouldDownloadIfNotDownloaded = true
						podFeed.PodcastEpisodes = append(podFeed.PodcastEpisodes, episode)
					}
				}
			}
		}
	}
	for _, episode := range podFeed.PodcastEpisodes {
		if episode.ShouldDownloadIfNotDownloaded && !episode.Downloaded() {
			_, filename := path.Split(episode.URL)
			episode.ShouldDownloadIfNotDownloaded = false
			go pogoutils.Download(episode.URL, "downloads/"+filename)
		}
	}
}

//Should be run concurrently. Will save all data to the configuration file
func (catcher *Catcher) SaveData() {
	b, jsonErr := json.MarshalIndent(catcher, "", "    ")
	if jsonErr == nil {
		file, fileErr := os.Create(catcher.ConfigLocation)
		if fileErr == nil {
			defer file.Close()
			file.Write(b)
			fmt.Println("Saved catcher data to", catcher.ConfigLocation)
		} else {
			fmt.Println("Error creating file", fileErr)
		}
	} else {
		fmt.Println("Error encoding JSON", jsonErr)
	}
}

//Should be run concurrently. Subscribe to a podcast feed
func (catcher *Catcher) AddPodcastFeed(feedURL string) {
	//Firstly check if the podcast feed has already been added
	for _, podcast := range catcher.Podcasts {
		if podcast.FeedURL == feedURL {
			return
		}
	}
	resp, err := http.Get(feedURL)
	if err == nil {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			var xmlResponse Fetched
			err := xml.Unmarshal(contents, &xmlResponse)
			if err == nil {
				catcher.AddPodcast(xmlResponse, feedURL)
			}
		}
	}

}

//Will add a podcast given by AddPodcastFeed. Do not call directly
func (catcher *Catcher) AddPodcast(xml Fetched, feedURL string) {
	podcast := catcher.getPodcastFromXML(xml, feedURL)
	mostRecentEpisode := podcast.PodcastEpisodes[0]
	mostRecentEpisode.ShouldDownloadIfNotDownloaded = true
	podcast.PodcastEpisodes[0] = mostRecentEpisode
	catcher.Podcasts = append(catcher.Podcasts, podcast)
	go catcher.SaveData()
	//Ensures that changes are reflected in the download queue
	catcher.addFeed <- podcast
}

//Gets a PodFeed object from some fetched XML
func (catcher *Catcher) getPodcastFromXML(xml Fetched, feedURL string) PodFeed {
	channel := xml.Channel
	podcast := PodFeed{}
	podcast.Name = channel.Title
	podcast.FeedURL = feedURL
	podcast.Site = channel.Link
	podcast.LastRefreshed = time.Now()
	podcast.Language = channel.Language
	podcast.Copyright = channel.Copyright
	podcast.Subtitle = channel.Subtitle
	podcast.Description = channel.Description
	podcast.Summary = channel.Summary
	podcast.Image = channel.Image.Href
	podcast.Categories = make([]string, 0)
	for _, category := range channel.Categories {
		var cat string
		if category.SubCategory.Text != "" {
			cat = category.Text + "/" + category.SubCategory.Text
		} else {
			cat = category.Text
		}
		podcast.Categories = append(podcast.Categories, cat)
	}
	podcast.Acronym = Acronym(podcast.Name)
	podcast.ID = catcher.UniqueIDForPodcast(podcast.Acronym)
	podcast.PodcastEpisodes = make([]PodEpisode, 0)

	for _, item := range channel.Items {
		episode := PodEpisode{}
		episode.Description = template.HTML(item.Description)
		episode.Title = item.Title
		episode.Author = item.Author
		episode.Image = item.Image.Href
		episode.PubDate = item.PubDate
		episode.URL = item.Enclosure.URL
		episode.Type = item.Enclosure.Type
		episode.Length = ParseDuration(item.Duration)
		podcast.PodcastEpisodes = append(podcast.PodcastEpisodes, episode)
	}
	return podcast
}

//Checks to see if an acronym is unique and if not appends a number
func (catcher *Catcher) UniqueIDForPodcast(podcastAcronym string) string {
	suffix := 1
	for _, podcast := range catcher.Podcasts {
		if podcast.Acronym == podcastAcronym {
			suffix++
		}
	}
	if suffix > 1 {
		return fmt.Sprintf("%s%d", podcastAcronym, suffix)
	}
	return podcastAcronym
}

//Gets an acronym for a string (Programming Thomas -> PT, I like Google -> ILG).
//Technically most of the generated 'acronyms' are actually initialisms because an acronym
//should be a real word, but I'm a programming langauge nerd, not an English langauge nerd.
func Acronym(original string) string {
	buf := bytes.NewBufferString("")
	textSplit := strings.Split(original, " ")
	for _, word := range textSplit {
		upperCase := strings.ToUpper(string(word[0]))
		buf.WriteString(upperCase)
	}
	return buf.String()
}

//Parses a time string into a duration (13:37 -> 13 * time.Minute + 37 * time.Second,
//however 10:20:30 -> 10 * time.Hour + 20 * time.Minute + 10 * time.Second)
func ParseDuration(dur string) time.Duration {
	split := strings.Split(dur, ":")
	var d int64
	d = 0
	//Has seconds component
	if len(split) >= 1 {
		t, _ := strconv.ParseInt(split[len(split)-1], 0, 0)
		d += t * int64(time.Second)
	}
	//Has minutes components
	if len(split) >= 2 {
		t, _ := strconv.ParseInt(split[len(split)-2], 0, 0)
		d += t * int64(time.Minute)
	}
	//Has hour component (unlikely?)
	if len(split) >= 3 {
		t, _ := strconv.ParseInt(split[len(split)-3], 0, 0)
		d += t * int64(time.Hour)
	}
	return time.Duration(d)
}

//Removes content in HTML tags
func (episode PodEpisode) PlainTextDescription() template.HTML {
	regex, _ := regexp.Compile("<[^>]*>")
	return template.HTML(regex.ReplaceAll([]byte(episode.Description), []byte("")))
}

//First 50 characters of the description
func (episode PodEpisode) PlainTextDescriptionBeginning() template.HTML {
	return template.HTML(episode.PlainTextDescription()[0:50]) + "..."
}

//Gets a date like 'Today' or 'Yesterday' to represent the date
func (episode PodEpisode) PubDateText() string {
	now := time.Now()
	then := episode.ReleaseDate()
	if now.Day() == then.Day() && now.Month() == then.Month() && now.Year() == then.Year() {
		return "Today"
	}
	yesterday := now.AddDate(0, 0, -1)
	if yesterday.Day() == then.Day() && yesterday.Month() == yesterday.Month() && now.Year() == then.Year() {
		return "Yesterday"
	}
	//Need to add locale
	return fmt.Sprintf("%d/%d/%d", then.Month(), then.Day(), then.Year())
}

//Parses the date from the XML data
func (episode PodEpisode) ReleaseDate() time.Time {
	then, err := time.Parse(time.RFC822Z, episode.PubDate)
	if err != nil {
		then, err = time.Parse(time.RFC1123Z, episode.PubDate)
		if err != nil {
			//Fail
			then = time.Now()
		}
	}
	return then
}

//Ensures that the current thread is using the most up to date version of the podcasts
//(resolves concurrency issues?!)
func (catcher *Catcher) UpdateAll() {
	fmt.Println("Update all")
	select {
	case podcasts := <-catcher.updatedPodcasts:
		catcher.Podcasts = podcasts
	default:
		break
	}
	fmt.Println("Exit update all")
}

//Determines whether or not this episode is an audio episode
func (episode PodEpisode) IsAudio() bool {
	return strings.HasPrefix(episode.Type, "audio")
}

//Determines whether or not this episode is a video episode
func (episode PodEpisode) IsVideo() bool {
	return strings.HasPrefix(episode.Type, "video")
}

//Determines whether or not the podcast has been downloaded
func (episode PodEpisode) Downloaded() bool {
	return pogoutils.FileExists(episode.DownloadedFilename())
}

//Gets the filename that the episode should be or has been downloaded at
func (episode PodEpisode) DownloadedFilename() string {
	_, filename := path.Split(episode.URL)
	return "downloads/" + filename
}
