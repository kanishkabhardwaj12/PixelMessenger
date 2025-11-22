import React, { useState } from 'react';
import { ArrowLeft, Upload, Image as ImageIcon, BarChart3 } from 'lucide-react';
import axios from 'axios';

function StegoAnalysis({ onBack }) {
  const [coverImage, setCoverImage] = useState(null);
  const [stegoImage, setStegoImage] = useState(null);
  const [analysis, setAnalysis] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleImageUpload = (e, type) => {
    const file = e.target.files[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        if (type === 'cover') {
          setCoverImage(e.target.result);
        } else {
          setStegoImage(e.target.result);
        }
        setAnalysis(null); // Reset analysis when new image is uploaded
      };
      reader.readAsDataURL(file);
    }
  };

  const analyzeImages = async () => {
    if (!coverImage || !stegoImage) {
      alert('Please upload both cover and stego images');
      return;
    }

    setLoading(true);

    try {
      // Convert data URLs to blobs
      const coverBlob = await fetch(coverImage).then(r => r.blob());
      const stegoBlob = await fetch(stegoImage).then(r => r.blob());

      // Create FormData for API call
      const formData = new FormData();
      formData.append('cover_image', coverBlob);
      formData.append('stego_image', stegoBlob);

      // Call backend analysis endpoint using axios
      const response = await axios.post('http://localhost:8082/analyze-stego', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });

      setAnalysis(response.data);
    } catch (error) {
      console.error('Analysis error:', error);
      alert('Error analyzing images: ' + (error.response?.data || error.message));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex items-center mb-8">
          <button
            onClick={onBack}
            className="flex items-center gap-2 px-4 py-2 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
            Back to Chat
          </button>
          <h1 className="text-3xl font-bold ml-6">Steganography Analysis</h1>
        </div>

        {/* Upload Section */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          {/* Cover Image Upload */}
          <div className="bg-gray-800 rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
              <ImageIcon className="h-5 w-5" />
              Cover Image (Original)
            </h2>
            <div className="border-2 border-dashed border-gray-600 rounded-lg p-4 mb-4">
              <input
                type="file"
                accept="image/*"
                onChange={(e) => handleImageUpload(e, 'cover')}
                className="hidden"
                id="cover-upload"
              />
              <label htmlFor="cover-upload" className="cursor-pointer flex flex-col items-center">
                {coverImage ? (
                  <img src={coverImage} alt="Cover" className="max-w-full max-h-48 rounded" />
                ) : (
                  <>
                    <Upload className="h-12 w-12 text-gray-400 mb-2" />
                    <p className="text-gray-400">Click to upload cover image</p>
                  </>
                )}
              </label>
            </div>
          </div>

          {/* Stego Image Upload */}
          <div className="bg-gray-800 rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
              <ImageIcon className="h-5 w-5" />
              Stego Image (Encoded)
            </h2>
            <div className="border-2 border-dashed border-gray-600 rounded-lg p-4 mb-4">
              <input
                type="file"
                accept="image/*"
                onChange={(e) => handleImageUpload(e, 'stego')}
                className="hidden"
                id="stego-upload"
              />
              <label htmlFor="stego-upload" className="cursor-pointer flex flex-col items-center">
                {stegoImage ? (
                  <img src={stegoImage} alt="Stego" className="max-w-full max-h-48 rounded" />
                ) : (
                  <>
                    <Upload className="h-12 w-12 text-gray-400 mb-2" />
                    <p className="text-gray-400">Click to upload stego image</p>
                  </>
                )}
              </label>
            </div>
          </div>
        </div>

        {/* Analyze Button */}
        <div className="text-center mb-8">
          <button
            onClick={analyzeImages}
            disabled={!coverImage || !stegoImage || loading}
            className="px-8 py-3 bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:bg-gray-600 disabled:cursor-not-allowed flex items-center gap-2 mx-auto"
          >
            <BarChart3 className="h-5 w-5" />
            {loading ? 'Analyzing...' : 'Analyze Steganography'}
          </button>
        </div>

        {/* Analysis Results */}
        {analysis && (
          <div className="bg-gray-800 rounded-lg p-6">
            <h2 className="text-2xl font-bold mb-6">Analysis Results</h2>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
              <div className="bg-gray-700 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-indigo-400">Total Pixel Changes</h3>
                <p className="text-2xl font-bold">{analysis.total_changes || 'N/A'}</p>
              </div>

              <div className="bg-gray-700 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-green-400">LSB Changes</h3>
                <p className="text-2xl font-bold">{analysis.lsb_changes || 'N/A'}</p>
              </div>

              <div className="bg-gray-700 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-yellow-400">Image Dimensions</h3>
                <p className="text-sm">{analysis.dimensions || 'N/A'}</p>
              </div>

              <div className="bg-gray-700 rounded-lg p-4">
                <h3 className="text-lg font-semibold text-red-400">Capacity Used</h3>
                <p className="text-lg">{analysis.capacity_used || 'N/A'}</p>
              </div>
            </div>

            {/* Sample Pixel Comparison */}
            {analysis.sample_pixels && (
              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-4">Sample Pixel Changes</h3>
                <div className="bg-gray-700 rounded-lg p-4 overflow-x-auto">
                  <pre className="text-sm text-gray-300 whitespace-pre-wrap">
                    {analysis.sample_pixels}
                  </pre>
                </div>
              </div>
            )}

            {/* Binary Representation */}
            {analysis.binary_comparison && (
              <div className="mb-6">
                <h3 className="text-xl font-semibold mb-4">Binary LSB Changes</h3>
                <div className="bg-gray-700 rounded-lg p-4 overflow-x-auto">
                  <pre className="text-sm text-gray-300 whitespace-pre-wrap">
                    {analysis.binary_comparison}
                  </pre>
                </div>
              </div>
            )}

            {/* Conclusion */}
            <div className="bg-indigo-900 rounded-lg p-4">
              <h3 className="text-xl font-semibold mb-2">Conclusion</h3>
              <p className="text-gray-200">
                {analysis.total_changes > 0
                  ? `✓ Steganography detected! ${analysis.lsb_changes} least significant bits have been modified to hide data.`
                  : '✗ No steganography changes detected between the images.'}
              </p>
            </div>
          </div>
        )}

        {/* Instructions */}
        <div className="bg-gray-800 rounded-lg p-6 mt-8">
          <h2 className="text-xl font-semibold mb-4">How It Works</h2>
          <div className="text-gray-300 space-y-2">
            <p>• <strong>Upload Cover Image:</strong> The original image before encoding</p>
            <p>• <strong>Upload Stego Image:</strong> The image with hidden message</p>
            <p>• <strong>Analysis:</strong> Compares pixel values to detect LSB modifications</p>
            <p>• <strong>LSB Steganography:</strong> Changes least significant bits to hide data invisibly</p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default StegoAnalysis;
