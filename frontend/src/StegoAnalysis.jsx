import React, { useState } from 'react';
import { ArrowLeft, Image as ImageIcon, Loader2, Search, Upload } from 'lucide-react';
import axios from 'axios';

const API_BASE_URL = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_API_BASE_URL
  ? import.meta.env.VITE_API_BASE_URL
  : 'http://localhost:8082';

function StegoAnalysis({ token, onBack }) {
  const [stegoImage, setStegoImage] = useState(null);
  const [stegoBlob, setStegoBlob] = useState(null);
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);

  const handleImageUpload = (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setStegoBlob(file);
    setResult(null);

    const reader = new FileReader();
    reader.onload = (event) => setStegoImage(event.target.result);
    reader.readAsDataURL(file);
  };

  const decodeImage = async () => {
    if (!stegoBlob) {
      alert('Please upload a stego image');
      return;
    }

    setLoading(true);
    setResult(null);

    try {
      const response = await axios.post(`${API_BASE_URL}/decode`, stegoBlob, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': stegoBlob.type || 'image/png',
        },
      });

      setResult(response.data);
    } catch (error) {
      console.error('Decode error:', error);
      alert('Error decoding image: ' + (error.response?.data || error.message));
    } finally {
      setLoading(false);
    }
  };

  const decodedText = result?.message || result?.message_base64 || '';
  const isBase64 = result?.encoding === 'base64' && !result?.message;

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center mb-8">
          <button
            onClick={onBack}
            className="flex items-center gap-2 px-4 py-2 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
            Back to Chat
          </button>
          <h1 className="text-3xl font-bold ml-6">Steganalysis</h1>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 mb-8">
          <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
            <ImageIcon className="h-5 w-5" />
            Stego Image
          </h2>
          <div className="border-2 border-dashed border-gray-600 rounded-lg p-4 mb-4">
            <input
              type="file"
              accept="image/*"
              onChange={handleImageUpload}
              className="hidden"
              id="stego-upload"
            />
            <label htmlFor="stego-upload" className="cursor-pointer flex flex-col items-center">
              {stegoImage ? (
                <img src={stegoImage} alt="Stego" className="max-w-full max-h-80 rounded" />
              ) : (
                <>
                  <Upload className="h-12 w-12 text-gray-400 mb-2" />
                  <p className="text-gray-400">Click to upload stego image</p>
                </>
              )}
            </label>
          </div>

          <button
            onClick={decodeImage}
            disabled={!stegoBlob || loading}
            className="px-6 py-3 bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:bg-gray-600 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {loading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Search className="h-5 w-5" />}
            {loading ? 'Decoding...' : 'Decode Hidden Text'}
          </button>
        </div>

        {result && (
          <div className="bg-gray-800 rounded-lg p-6">
            <h2 className="text-2xl font-bold mb-4">Decoded Message</h2>
            <div className="bg-gray-900 border border-gray-700 rounded-lg p-4">
              <pre className="text-gray-100 whitespace-pre-wrap break-words">{decodedText || 'No message found'}</pre>
            </div>
            {isBase64 && (
              <p className="mt-3 text-sm text-yellow-300">
                Decoded bytes are not valid UTF-8, so they are shown as base64.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default StegoAnalysis;
